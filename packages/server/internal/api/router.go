package api

import (
	"net/http"

	"co-review/server/internal/skills"
	skillassets "co-review/server/skills"
)

// NewRouter creates the server HTTP routing tree.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /api/v1/skills", skillsHandler)
	mux.HandleFunc("/api/v1/", apiNotFoundHandler)
	mux.HandleFunc("/", rootNotFoundHandler)
	return mux
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
