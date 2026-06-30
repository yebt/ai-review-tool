package repos

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"co-review/server/internal/db"
)

func TestServiceRepoCRUDAndModelConfig(t *testing.T) {
	service := testService(t)
	ctx := context.Background()

	repo, err := service.Create(ctx, RepoInput{Name: "acme/widget", URL: "https://gitlab.com/acme/widget", Platform: "gitlab"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.DefaultBranch != "main" || !repo.Active || repo.PublishMode != PublishModeSequential {
		t.Fatalf("repo defaults mismatch: %+v", repo)
	}

	patchedName := "acme/widget-renamed"
	active := false
	repo, err = service.Patch(ctx, repo.ID, RepoPatch{Name: &patchedName, Active: &active})
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}
	if repo.Name != patchedName || repo.Active {
		t.Fatalf("patched repo mismatch: %+v", repo)
	}

	cfg, err := service.PutModel(ctx, repo.ID, ModelInput{Provider: "openai", ModelName: "gpt-4.1", APIKeyEnv: "OPENAI_API_KEY"})
	if err != nil {
		t.Fatalf("PutModel() error = %v", err)
	}
	if cfg.APIKeyEnv != "OPENAI_API_KEY" || cfg.ModelName != "gpt-4.1" {
		t.Fatalf("model config mismatch: %+v", cfg)
	}
	if _, err := service.PutModel(ctx, repo.ID, ModelInput{Provider: "openai", ModelName: "gpt-4.1", APIKeyEnv: "sk-not-an-env-name"}); !IsInvalidInput(err) {
		t.Fatalf("PutModel() raw-looking key error = %v, want invalid input", err)
	}

	if err := service.Delete(ctx, repo.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(ctx, repo.ID); !IsNotFound(err) {
		t.Fatalf("Get() after delete error = %v, want not found", err)
	}
}

func TestInferGitLabFallsBackToMainWithoutNetwork(t *testing.T) {
	service := testService(t)
	inference, err := service.InferGitLab("git@gitlab.com:acme/widget.git")
	if err != nil {
		t.Fatalf("InferGitLab() error = %v", err)
	}
	if inference.ProjectPath != "acme/widget" || inference.DefaultBranch != "main" || inference.DefaultBranchSource != "fallback" {
		t.Fatalf("inference mismatch: %+v", inference)
	}
}

func TestMemoryDefaultsAndPromptContext(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	repo, err := service.Create(ctx, RepoInput{Name: "acme/widget", URL: "https://gitlab.com/acme/widget", Platform: "gitlab"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	decision, err := service.AddMemory(ctx, repo.ID, MemoryInput{Type: MemoryTypeAcceptedDecision, Key: "prefer-table-tests", Content: "Prefer table-driven tests for parser behavior.", Dimension: "reliability", SourceMR: "7"})
	if err != nil {
		t.Fatalf("AddMemory(decision) error = %v", err)
	}
	if decision.ExpiresAt != nil {
		t.Fatalf("accepted decision expires_at = %v, want nil", decision.ExpiresAt)
	}
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	if _, err := service.AddMemory(ctx, repo.ID, MemoryInput{Type: MemoryTypeCodebasePattern, Key: "errors", Content: "API errors use structured JSON envelopes.", ExpiresAt: expires}); err != nil {
		t.Fatalf("AddMemory(pattern) error = %v", err)
	}

	contextText, err := service.RenderPromptContext(ctx, repo.ID)
	if err != nil {
		t.Fatalf("RenderPromptContext() error = %v", err)
	}
	for _, want := range []string{"Repository memory context:", "Accepted decisions:", "[prefer-table-tests] Prefer table-driven tests", "Known codebase patterns:", "[errors] API errors"} {
		if !strings.Contains(contextText, want) {
			t.Fatalf("context %q missing %q", contextText, want)
		}
	}
}

func TestMemoryUpsertPreventsDuplicateLogicalKeys(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	repo, err := service.Create(ctx, RepoInput{Name: "acme/widget", URL: "https://gitlab.com/acme/widget", Platform: "gitlab"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := service.AddMemory(ctx, repo.ID, MemoryInput{Type: MemoryTypeCodebasePattern, Key: "api-errors", Content: "Use JSON API errors."}); err != nil {
		t.Fatalf("AddMemory(first) error = %v", err)
	}
	if _, err := service.AddMemory(ctx, repo.ID, MemoryInput{Type: MemoryTypeCodebasePattern, Key: "api-errors", Content: "Use structured JSON API errors."}); err != nil {
		t.Fatalf("AddMemory(second) error = %v", err)
	}

	memory, err := service.ListMemory(ctx, repo.ID)
	if err != nil {
		t.Fatalf("ListMemory() error = %v", err)
	}
	if len(memory) != 1 || memory[0].Content != "Use structured JSON API errors." {
		t.Fatalf("memory upsert mismatch: %+v", memory)
	}
}

func testService(t *testing.T) *Service {
	t.Helper()
	return &Service{Repo: NewRepository(testDB(t))}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return database
}
