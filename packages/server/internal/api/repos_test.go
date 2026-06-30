package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"co-review/server/internal/repos"
)

func TestReposAPICRUDModelInferenceAndMemory(t *testing.T) {
	repoService := &repos.Service{Repo: repos.NewRepository(reviewsTestDB(t))}
	router := NewRouterWithDeps(RouterDeps{Repos: repoService})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repos/infer", strings.NewReader(`{"url":"https://gitlab.com/acme/widget.git"}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("infer status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"default_branch_source":"fallback"`) {
		t.Fatalf("infer response missing fallback source: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/repos", strings.NewReader(`{"name":"acme/widget","url":"https://gitlab.com/acme/widget","platform":"gitlab","default_branch":"main","auto_publish":false,"publish_mode":"sequential","active":true}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Repo repos.Repo `json:"repo"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode repo: %v", err)
	}

	paths := []string{"/api/v1/repos", "/api/v1/repos/" + created.Repo.ID}
	for _, path := range paths {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", path, rec.Code, rec.Body.String())
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/repos/"+created.Repo.ID, strings.NewReader(`{"active":false}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"active":false`) {
		t.Fatalf("patch response = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/repos/"+created.Repo.ID+"/model", strings.NewReader(`{"provider":"anthropic","model_name":"claude-test","api_key_env":"ANTHROPIC_API_KEY"}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "sk-") {
		t.Fatalf("put model response = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/repos/"+created.Repo.ID+"/model", strings.NewReader(`{"provider":"anthropic","model_name":"claude-test","api_key_env":"sk-not-an-env-name"}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("raw-looking key status = %d, want 400: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/repos/"+created.Repo.ID+"/memory", strings.NewReader(`{"type":"accepted_decision","key":"testing","content":"Keep tests deterministic."}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add memory status = %d: %s", rec.Code, rec.Body.String())
	}
	var memoryCreated struct {
		Memory repos.MemoryEntry `json:"memory"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &memoryCreated); err != nil {
		t.Fatalf("decode memory: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/repos/"+created.Repo.ID+"/memory", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Keep tests deterministic") {
		t.Fatalf("list memory response = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/repos/"+created.Repo.ID+"/memory/"+memoryCreated.Memory.ID, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete memory status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/repos/"+created.Repo.ID, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete repo status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestReposAPIRejectsInvalidInferenceURL(t *testing.T) {
	repoService := &repos.Service{Repo: repos.NewRepository(reviewsTestDB(t))}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repos/infer", strings.NewReader(`{"url":"not a gitlab url"}`))

	NewRouterWithDeps(RouterDeps{Repos: repoService}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("infer status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}
