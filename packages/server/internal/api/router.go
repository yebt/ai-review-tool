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
	writeJSON(w, http.StatusOK, map[string]any{"skills": loaded})
}

func rootNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "Route not found", r.URL.Path)
}
