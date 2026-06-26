package reviews

import (
	"encoding/json"
	"time"
)

const (
	StatusPending          = "pending"
	StatusRunning          = "running"
	StatusGenerated        = "generated"
	StatusAwaitingApproval = "awaiting_approval"
	StatusError            = "error"

	CommentStatusPending = "pending"
)

type CreateRequest struct {
	Platform    string `json:"platform"`
	ProjectURL  string `json:"project_url"`
	ProjectPath string `json:"project_path"`
	MRIID       int    `json:"mr_iid"`
}

type Review struct {
	ID          string          `json:"id"`
	RepoID      string          `json:"repo_id"`
	ProjectPath string          `json:"project_path"`
	ProjectURL  string          `json:"project_url"`
	Platform    string          `json:"platform"`
	MRID        string          `json:"mr_id"`
	MRURL       string          `json:"mr_url"`
	MRTitle     string          `json:"mr_title"`
	BaseSHA     string          `json:"base_sha"`
	StartSHA    string          `json:"start_sha"`
	HeadSHA     string          `json:"head_sha"`
	Status      string          `json:"status"`
	Scores      json.RawMessage `json:"scores,omitempty"`
	Verdict     string          `json:"verdict,omitempty"`
	ModelUsed   string          `json:"model_used"`
	Error       json.RawMessage `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Comments    []Comment       `json:"comments,omitempty"`
}

type Comment struct {
	ID                string    `json:"id"`
	ReviewID          string    `json:"review_id"`
	Dimension         string    `json:"dimension"`
	Severity          string    `json:"severity"`
	File              string    `json:"file"`
	LineStart         *int      `json:"line_start,omitempty"`
	LineEnd           *int      `json:"line_end,omitempty"`
	Evidence          string    `json:"evidence"`
	Why               string    `json:"why"`
	SuggestionSnippet string    `json:"suggestion_snippet,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

type HarnessError struct {
	Dimension string `json:"dimension"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}
