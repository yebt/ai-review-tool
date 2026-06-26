package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	NewRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %q, want ok", body["status"])
	}
}

func TestAPINotFoundHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repos", nil)
	rec := httptest.NewRecorder()

	NewRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want not_found", body.Error.Code)
	}
}

func TestSkillsHandlerReturnsEmbeddedSkillsOutsideServerCWD(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	rec := httptest.NewRecorder()

	NewRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Skills []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Dimension   string `json:"dimension"`
			Model       string `json:"model"`
			Body        string `json:"body"`
			FilePath    string `json:"filePath"`
			Harness     struct {
				TimeoutSeconds int    `json:"timeout_seconds"`
				OutputSchema   string `json:"output_schema"`
			} `json:"harness"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Skills) != 4 {
		t.Fatalf("skills count = %d, want 4", len(body.Skills))
	}

	want := map[string]string{
		"review-readability": "readability",
		"review-reliability": "reliability",
		"review-resilience":  "resilience",
		"review-risk":        "risk",
	}
	for _, skill := range body.Skills {
		if want[skill.Name] != skill.Dimension {
			t.Fatalf("skill %q dimension = %q, want %q", skill.Name, skill.Dimension, want[skill.Name])
		}
		if skill.Description == "" || skill.Model == "" {
			t.Fatalf("skill %q missing public metadata: %+v", skill.Name, skill)
		}
		if skill.Harness.TimeoutSeconds == 0 || skill.Harness.OutputSchema == "" {
			t.Fatalf("skill %q missing harness metadata: %+v", skill.Name, skill.Harness)
		}
		if skill.Body != "" {
			t.Fatalf("skill %q leaked prompt body", skill.Name)
		}
		if skill.FilePath != "" {
			t.Fatalf("skill %q leaked internal file path %q", skill.Name, skill.FilePath)
		}
		delete(want, skill.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing skills: %v", want)
	}
}
