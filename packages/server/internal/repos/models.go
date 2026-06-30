package repos

import "time"

const (
	PlatformGitLab = "gitlab"

	PublishModeSequential = "sequential"
	PublishModeAll        = "all"

	MemoryTypeAcceptedDecision = "accepted_decision"
	MemoryTypeCodebasePattern  = "codebase_pattern"
)

type Repo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Platform      string    `json:"platform"`
	DefaultBranch string    `json:"default_branch"`
	AutoPublish   bool      `json:"auto_publish"`
	PublishMode   string    `json:"publish_mode"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
}

type RepoInput struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Platform      string `json:"platform"`
	DefaultBranch string `json:"default_branch"`
	AutoPublish   *bool  `json:"auto_publish"`
	PublishMode   string `json:"publish_mode"`
	Active        *bool  `json:"active"`
}

type RepoPatch struct {
	Name          *string `json:"name"`
	URL           *string `json:"url"`
	Platform      *string `json:"platform"`
	DefaultBranch *string `json:"default_branch"`
	AutoPublish   *bool   `json:"auto_publish"`
	PublishMode   *string `json:"publish_mode"`
	Active        *bool   `json:"active"`
}

type InferRequest struct {
	URL string `json:"url"`
}

type Inference struct {
	Name                string `json:"name"`
	URL                 string `json:"url"`
	Platform            string `json:"platform"`
	DefaultBranch       string `json:"default_branch"`
	DefaultBranchSource string `json:"default_branch_source"`
	ProjectPath         string `json:"project_path"`
	Host                string `json:"host"`
}

type ModelConfig struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Provider  string    `json:"provider"`
	ModelName string    `json:"model_name"`
	APIKeyEnv string    `json:"api_key_env,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

type ModelInput struct {
	Provider  string `json:"provider"`
	ModelName string `json:"model_name"`
	APIKeyEnv string `json:"api_key_env"`
}

type MemoryEntry struct {
	ID        string     `json:"id"`
	RepoID    string     `json:"repo_id"`
	Type      string     `json:"type"`
	Key       string     `json:"key"`
	Content   string     `json:"content"`
	Dimension string     `json:"dimension,omitempty"`
	SourceMR  string     `json:"source_mr,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type MemoryInput struct {
	Type      string `json:"type"`
	Key       string `json:"key"`
	Content   string `json:"content"`
	Dimension string `json:"dimension"`
	SourceMR  string `json:"source_mr"`
	ExpiresAt string `json:"expires_at"`
}
