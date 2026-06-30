package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"co-review/server/internal/events"
	"co-review/server/internal/repos"
	"co-review/server/internal/reviews"
	"co-review/server/internal/skills"
	skillassets "co-review/server/skills"
)

type RouterDeps struct {
	Reviews *reviews.Service
	Repos   *repos.Service
	Broker  *events.Broker
}

// NewRouter creates the server HTTP routing tree.
func NewRouter() http.Handler {
	return NewRouterWithDeps(RouterDeps{})
}

func NewRouterWithDeps(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /api/v1/skills", skillsHandler)
	if deps.Repos != nil {
		mux.HandleFunc("POST /api/v1/repos", createRepoHandler(deps.Repos))
		mux.HandleFunc("GET /api/v1/repos", listReposHandler(deps.Repos))
		mux.HandleFunc("POST /api/v1/repos/infer", inferRepoHandler(deps.Repos))
		mux.HandleFunc("GET /api/v1/repos/", repoSubresourceHandler(deps.Repos))
		mux.HandleFunc("POST /api/v1/repos/", repoSubresourceHandler(deps.Repos))
		mux.HandleFunc("PATCH /api/v1/repos/", repoSubresourceHandler(deps.Repos))
		mux.HandleFunc("DELETE /api/v1/repos/", repoSubresourceHandler(deps.Repos))
		mux.HandleFunc("PUT /api/v1/repos/", repoSubresourceHandler(deps.Repos))
	}
	mux.HandleFunc("POST /api/v1/reviews", createReviewHandler(deps.Reviews))
	mux.HandleFunc("GET /api/v1/reviews", listReviewsHandler(deps.Reviews))
	mux.HandleFunc("GET /api/v1/reviews/", reviewSubresourceHandler(deps.Reviews, deps.Broker))
	mux.HandleFunc("PATCH /api/v1/reviews/", reviewSubresourceHandler(deps.Reviews, deps.Broker))
	mux.HandleFunc("/api/v1/", apiNotFoundHandler)
	mux.HandleFunc("/", rootNotFoundHandler)
	return mux
}

func createReviewHandler(service *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "reviews_not_configured", "Review service is not configured", r.URL.Path)
			return
		}
		var req reviews.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
			return
		}
		review, err := service.Create(r.Context(), req)
		if err != nil {
			if reviews.IsInvalidInput(err) {
				writeError(w, http.StatusBadRequest, "invalid_review_request", err.Error(), r.URL.Path)
				return
			}
			if review.ID != "" {
				writeJSON(w, http.StatusAccepted, map[string]any{"review": review, "error": err.Error()})
				return
			}
			writeError(w, http.StatusBadGateway, "review_create_failed", err.Error(), r.URL.Path)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"review": review})
	}
}

func listReviewsHandler(service *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "reviews_not_configured", "Review service is not configured", r.URL.Path)
			return
		}
		items, err := service.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "reviews_list_failed", err.Error(), r.URL.Path)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reviews": items})
	}
}

func reviewSubresourceHandler(service *reviews.Service, broker *events.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "reviews_not_configured", "Review service is not configured", r.URL.Path)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/reviews/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			apiNotFoundHandler(w, r)
			return
		}
		reviewID := parts[0]
		if len(parts) == 1 && r.Method == http.MethodGet {
			review, err := service.Get(r.Context(), reviewID)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "review_not_found", "Review not found", r.URL.Path)
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "review_get_failed", err.Error(), r.URL.Path)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"review": review})
			return
		}
		if len(parts) == 2 && parts[1] == "comments" && r.Method == http.MethodGet {
			comments, err := service.Comments(r.Context(), reviewID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "review_comments_failed", err.Error(), r.URL.Path)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
			return
		}
		if len(parts) == 3 && parts[1] == "comments" && r.Method == http.MethodPatch {
			var req reviews.CommentStatusUpdate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
				return
			}
			comment, err := service.UpdateCommentStatus(r.Context(), reviewID, parts[2], req.Status)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "comment_not_found", "Comment not found", r.URL.Path)
				return
			}
			if reviews.IsInvalidInput(err) {
				writeError(w, http.StatusBadRequest, "invalid_comment_status", err.Error(), r.URL.Path)
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "comment_update_failed", err.Error(), r.URL.Path)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"comment": comment})
			return
		}
		if len(parts) == 2 && parts[1] == "events" && r.Method == http.MethodGet {
			serveReviewEvents(w, r, broker, reviewID)
			return
		}
		apiNotFoundHandler(w, r)
	}
}

func serveReviewEvents(w http.ResponseWriter, r *http.Request, broker *events.Broker, reviewID string) {
	if broker == nil {
		writeError(w, http.StatusNotImplemented, "events_not_configured", "Review events are not configured", r.URL.Path)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "sse_not_supported", "Streaming is not supported", r.URL.Path)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, unsubscribe := broker.Subscribe(reviewID)
	defer unsubscribe()
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write([]byte("event: " + event.Name + "\n"))
			_, _ = w.Write([]byte("data: " + string(event.Data) + "\n\n"))
			flusher.Flush()
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func apiNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "API route not found", r.URL.Path)
}

func skillsHandler(w http.ResponseWriter, r *http.Request) {
	loaded, err := skills.LoadFS(skillassets.FS, ".")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "skills_load_failed", "Could not load embedded skills", r.URL.Path)
		return
	}
	response := make([]skillResponse, 0, len(loaded))
	for _, skill := range loaded {
		response = append(response, skillResponse{
			Name:        skill.Name,
			Description: skill.Description,
			Dimension:   skill.Dimension,
			Model:       skill.Model,
			Readonly:    skill.Readonly,
			Background:  skill.Background,
			Harness: skillHarnessResponse{
				TimeoutSeconds:     skill.Harness.TimeoutSeconds,
				MaxRetries:         skill.Harness.MaxRetries,
				OutputSchema:       skill.Harness.OutputSchema,
				RequireEvidence:    skill.Harness.RequireEvidence,
				MinFindingsQuality: skill.Harness.MinFindingsQuality,
			},
			Memory: skillMemoryResponse{
				InjectContext: skill.Memory.InjectContext,
				SaveFindings:  skill.Memory.SaveFindings,
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"skills": response})
}

type skillResponse struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Dimension   string               `json:"dimension"`
	Model       string               `json:"model"`
	Readonly    bool                 `json:"readonly"`
	Background  bool                 `json:"background"`
	Harness     skillHarnessResponse `json:"harness"`
	Memory      skillMemoryResponse  `json:"memory"`
}

type skillHarnessResponse struct {
	TimeoutSeconds     int    `json:"timeout_seconds"`
	MaxRetries         int    `json:"max_retries"`
	OutputSchema       string `json:"output_schema"`
	RequireEvidence    bool   `json:"require_evidence"`
	MinFindingsQuality string `json:"min_findings_quality"`
}

type skillMemoryResponse struct {
	InjectContext bool `json:"inject_context"`
	SaveFindings  bool `json:"save_findings"`
}

func rootNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "Route not found", r.URL.Path)
}
