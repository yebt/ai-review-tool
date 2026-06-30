package repos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"co-review/server/internal/platform"
)

type Service struct{ Repo *Repository }

func (s *Service) Create(ctx context.Context, input RepoInput) (Repo, error) {
	repo, err := repoFromInput(input)
	if err != nil {
		return Repo{}, err
	}
	repo.ID = stableID("repo", repo.Platform+":"+repo.Name)
	if err := s.Repo.CreateRepo(ctx, repo); err != nil {
		return Repo{}, err
	}
	return s.Repo.GetRepo(ctx, repo.ID)
}

func (s *Service) Ensure(ctx context.Context, input RepoInput) (Repo, error) {
	repo, err := repoFromInput(input)
	if err != nil {
		return Repo{}, err
	}
	repo.ID = stableID("repo", repo.Platform+":"+repo.Name)
	if err := s.Repo.UpsertRepo(ctx, repo); err != nil {
		return Repo{}, err
	}
	return s.Repo.GetRepo(ctx, repo.ID)
}

func (s *Service) List(ctx context.Context) ([]Repo, error)         { return s.Repo.ListRepos(ctx) }
func (s *Service) Get(ctx context.Context, id string) (Repo, error) { return s.Repo.GetRepo(ctx, id) }

func (s *Service) Patch(ctx context.Context, id string, patch RepoPatch) (Repo, error) {
	repo, err := s.Repo.GetRepo(ctx, id)
	if err != nil {
		return Repo{}, err
	}
	if patch.Name != nil {
		repo.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.URL != nil {
		repo.URL = strings.TrimSpace(*patch.URL)
	}
	if patch.Platform != nil {
		repo.Platform = strings.ToLower(strings.TrimSpace(*patch.Platform))
	}
	if patch.DefaultBranch != nil {
		repo.DefaultBranch = strings.TrimSpace(*patch.DefaultBranch)
	}
	if patch.AutoPublish != nil {
		repo.AutoPublish = *patch.AutoPublish
	}
	if patch.PublishMode != nil {
		repo.PublishMode = strings.TrimSpace(*patch.PublishMode)
	}
	if patch.Active != nil {
		repo.Active = *patch.Active
	}
	if err := validateRepo(repo); err != nil {
		return Repo{}, err
	}
	if err := s.Repo.UpdateRepo(ctx, repo); err != nil {
		return Repo{}, err
	}
	return s.Repo.GetRepo(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error { return s.Repo.DeleteRepo(ctx, id) }

func (s *Service) InferGitLab(rawURL string) (Inference, error) {
	project, err := platform.InferGitLabProject(rawURL)
	if err != nil {
		return Inference{}, err
	}
	return Inference{Name: project.Path, URL: project.WebURL, Platform: PlatformGitLab, DefaultBranch: "main", DefaultBranchSource: "fallback", ProjectPath: project.Path, Host: project.Host}, nil
}

func (s *Service) PutModel(ctx context.Context, repoID string, input ModelInput) (ModelConfig, error) {
	if _, err := s.Repo.GetRepo(ctx, repoID); err != nil {
		return ModelConfig{}, err
	}
	provider := strings.TrimSpace(input.Provider)
	model := strings.TrimSpace(input.ModelName)
	apiKeyEnv := strings.TrimSpace(input.APIKeyEnv)
	if provider == "" || model == "" {
		return ModelConfig{}, errInvalidInput("provider and model_name are required")
	}
	if apiKeyEnv != "" && !isEnvVarName(apiKeyEnv) {
		return ModelConfig{}, errInvalidInput("api_key_env must be an environment variable name")
	}
	cfg := ModelConfig{ID: stableID("model", fmt.Sprintf("%s:%s:%s:%d", repoID, provider, model, time.Now().UnixNano())), RepoID: repoID, Provider: provider, ModelName: model, APIKeyEnv: apiKeyEnv, IsActive: true}
	if err := s.Repo.PutModel(ctx, cfg); err != nil {
		return ModelConfig{}, err
	}
	return s.Repo.GetActiveModel(ctx, repoID)
}

func (s *Service) GetModel(ctx context.Context, repoID string) (ModelConfig, error) {
	if _, err := s.Repo.GetRepo(ctx, repoID); err != nil {
		return ModelConfig{}, err
	}
	return s.Repo.GetActiveModel(ctx, repoID)
}

func (s *Service) AddMemory(ctx context.Context, repoID string, input MemoryInput) (MemoryEntry, error) {
	if _, err := s.Repo.GetRepo(ctx, repoID); err != nil {
		return MemoryEntry{}, err
	}
	entry, err := MemoryFromInput(repoID, input)
	if err != nil {
		return MemoryEntry{}, err
	}
	if err := s.Repo.CreateMemory(ctx, entry); err != nil {
		return MemoryEntry{}, err
	}
	items, err := s.Repo.ListMemory(ctx, repoID)
	if err != nil {
		return MemoryEntry{}, err
	}
	for _, item := range items {
		if item.ID == entry.ID || (item.Type == entry.Type && item.Key == entry.Key) {
			return item, nil
		}
	}
	return entry, nil
}

func (s *Service) ListMemory(ctx context.Context, repoID string) ([]MemoryEntry, error) {
	if _, err := s.Repo.GetRepo(ctx, repoID); err != nil {
		return nil, err
	}
	return s.Repo.ListMemory(ctx, repoID)
}

func (s *Service) DeleteMemory(ctx context.Context, repoID, memoryID string) error {
	return s.Repo.DeleteMemory(ctx, repoID, memoryID)
}

func (s *Service) RenderPromptContext(ctx context.Context, repoID string) (string, error) {
	items, err := s.Repo.ListMemory(ctx, repoID)
	if err != nil {
		return "", err
	}
	return RenderPromptContext(items), nil
}

func repoFromInput(input RepoInput) (Repo, error) {
	repo := Repo{Name: strings.TrimSpace(input.Name), URL: strings.TrimSpace(input.URL), Platform: strings.ToLower(strings.TrimSpace(input.Platform)), DefaultBranch: strings.TrimSpace(input.DefaultBranch), PublishMode: strings.TrimSpace(input.PublishMode), Active: true}
	if repo.Platform == "" {
		repo.Platform = PlatformGitLab
	}
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}
	if repo.PublishMode == "" {
		repo.PublishMode = PublishModeSequential
	}
	if input.AutoPublish != nil {
		repo.AutoPublish = *input.AutoPublish
	}
	if input.Active != nil {
		repo.Active = *input.Active
	}
	return repo, validateRepo(repo)
}

func validateRepo(repo Repo) error {
	if repo.Name == "" || repo.URL == "" {
		return errInvalidInput("name and url are required")
	}
	if repo.Platform != PlatformGitLab {
		return errInvalidInput("only gitlab repos are supported")
	}
	if repo.DefaultBranch == "" {
		return errInvalidInput("default_branch is required")
	}
	if repo.PublishMode != PublishModeSequential && repo.PublishMode != PublishModeAll {
		return errInvalidInput("publish_mode must be sequential or all")
	}
	return nil
}

func MemoryFromInput(repoID string, input MemoryInput) (MemoryEntry, error) {
	typ := strings.TrimSpace(input.Type)
	key := strings.TrimSpace(input.Key)
	content := strings.TrimSpace(input.Content)
	if typ != MemoryTypeAcceptedDecision && typ != MemoryTypeCodebasePattern {
		return MemoryEntry{}, errInvalidInput("memory type must be accepted_decision or codebase_pattern")
	}
	if content == "" {
		return MemoryEntry{}, errInvalidInput("memory content is required")
	}
	if key == "" {
		key = path.Base(content)
		if len(key) > 80 {
			key = key[:80]
		}
	}
	var expires *time.Time
	if strings.TrimSpace(input.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.ExpiresAt))
		if err != nil {
			return MemoryEntry{}, errInvalidInput("expires_at must be RFC3339")
		}
		expires = &parsed
	}
	return MemoryEntry{ID: stableID("mem", fmt.Sprintf("%s:%s:%s:%d", repoID, typ, key, time.Now().UnixNano())), RepoID: repoID, Type: typ, Key: key, Content: content, Dimension: strings.TrimSpace(input.Dimension), SourceMR: strings.TrimSpace(input.SourceMR), ExpiresAt: expires}, nil
}

func RenderPromptContext(items []MemoryEntry) string {
	var decisions, patterns []MemoryEntry
	for _, item := range items {
		switch item.Type {
		case MemoryTypeAcceptedDecision:
			decisions = append(decisions, item)
		case MemoryTypeCodebasePattern:
			patterns = append(patterns, item)
		}
	}
	if len(decisions) == 0 && len(patterns) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Repository memory context:\n")
	if len(decisions) > 0 {
		b.WriteString("Accepted decisions:\n")
		for _, item := range decisions {
			b.WriteString(fmt.Sprintf("- [%s] %s\n", item.Key, item.Content))
		}
	}
	if len(patterns) > 0 {
		b.WriteString("Known codebase patterns:\n")
		for _, item := range patterns {
			b.WriteString(fmt.Sprintf("- [%s] %s\n", item.Key, item.Content))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func isEnvVarName(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && r != '_' {
				return false
			}
			continue
		}
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

type invalidInputError struct{ message string }

func (e invalidInputError) Error() string  { return e.message }
func errInvalidInput(message string) error { return invalidInputError{message: message} }
func IsInvalidInput(err error) bool        { var target invalidInputError; return errors.As(err, &target) }
func IsNotFound(err error) bool            { return errors.Is(err, sql.ErrNoRows) }
