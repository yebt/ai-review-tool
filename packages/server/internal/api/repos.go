package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"co-review/server/internal/repos"
)

func createRepoHandler(service *repos.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "repos_not_configured", "Repo service is not configured", r.URL.Path)
			return
		}
		var req repos.RepoInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
			return
		}
		repo, err := service.Create(r.Context(), req)
		writeRepoResult(w, r, repo, err, http.StatusCreated)
	}
}

func listReposHandler(service *repos.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "repos_not_configured", "Repo service is not configured", r.URL.Path)
			return
		}
		items, err := service.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "repos_list_failed", err.Error(), r.URL.Path)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
	}
}

func inferRepoHandler(service *repos.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "repos_not_configured", "Repo service is not configured", r.URL.Path)
			return
		}
		var req repos.InferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
			return
		}
		inference, err := service.InferGitLab(req.URL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "repo_infer_failed", err.Error(), r.URL.Path)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repo": inference})
	}
}

func repoSubresourceHandler(service *repos.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusNotImplemented, "repos_not_configured", "Repo service is not configured", r.URL.Path)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" || parts[0] == "infer" {
			apiNotFoundHandler(w, r)
			return
		}
		repoID := parts[0]
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				repo, err := service.Get(r.Context(), repoID)
				writeRepoResult(w, r, repo, err, http.StatusOK)
			case http.MethodPatch:
				var req repos.RepoPatch
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
					return
				}
				repo, err := service.Patch(r.Context(), repoID, req)
				writeRepoResult(w, r, repo, err, http.StatusOK)
			case http.MethodDelete:
				if err := service.Delete(r.Context(), repoID); err != nil {
					writeRepoError(w, r, err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				apiNotFoundHandler(w, r)
			}
			return
		}
		if len(parts) == 2 && parts[1] == "model" {
			switch r.Method {
			case http.MethodGet:
				cfg, err := service.GetModel(r.Context(), repoID)
				if err != nil {
					writeRepoError(w, r, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"model": cfg})
			case http.MethodPut:
				var req repos.ModelInput
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
					return
				}
				cfg, err := service.PutModel(r.Context(), repoID, req)
				if err != nil {
					writeRepoError(w, r, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"model": cfg})
			default:
				apiNotFoundHandler(w, r)
			}
			return
		}
		if len(parts) == 2 && parts[1] == "memory" {
			switch r.Method {
			case http.MethodGet:
				items, err := service.ListMemory(r.Context(), repoID)
				if err != nil {
					writeRepoError(w, r, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"memory": items})
			case http.MethodPost:
				var req repos.MemoryInput
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", r.URL.Path)
					return
				}
				entry, err := service.AddMemory(r.Context(), repoID, req)
				if err != nil {
					writeRepoError(w, r, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]any{"memory": entry})
			default:
				apiNotFoundHandler(w, r)
			}
			return
		}
		if len(parts) == 3 && parts[1] == "memory" && r.Method == http.MethodDelete {
			if err := service.DeleteMemory(r.Context(), repoID, parts[2]); err != nil {
				writeRepoError(w, r, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		apiNotFoundHandler(w, r)
	}
}

func writeRepoResult(w http.ResponseWriter, r *http.Request, repo repos.Repo, err error, status int) {
	if err != nil {
		writeRepoError(w, r, err)
		return
	}
	writeJSON(w, status, map[string]any{"repo": repo})
}

func writeRepoError(w http.ResponseWriter, r *http.Request, err error) {
	if err == sql.ErrNoRows || repos.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "repo_not_found", "Repo not found", r.URL.Path)
		return
	}
	if repos.IsInvalidInput(err) {
		writeError(w, http.StatusBadRequest, "invalid_repo_request", err.Error(), r.URL.Path)
		return
	}
	writeError(w, http.StatusInternalServerError, "repo_request_failed", err.Error(), r.URL.Path)
}
