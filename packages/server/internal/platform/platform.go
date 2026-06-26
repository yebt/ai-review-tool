package platform

import "context"

// PlatformClient is the smallest Phase 3 boundary needed to infer repositories
// and fetch merge request context. Publishing inline comments is intentionally
// left out until the approval/publish phase.
type PlatformClient interface {
	InferProject(rawURL string) (ProjectIdentity, error)
	FetchMergeRequestContext(ctx context.Context, projectPath string, mrIID int) (MergeRequestContext, error)
}

// ProjectIdentity is a provider-neutral project reference.
type ProjectIdentity struct {
	Platform string `json:"platform"`
	Host     string `json:"host"`
	Path     string `json:"path"`
	WebURL   string `json:"web_url"`
}

// MergeRequestMetadata contains review-facing MR details.
type MergeRequestMetadata struct {
	IID    int    `json:"iid"`
	Title  string `json:"title"`
	WebURL string `json:"web_url"`
}

// MergeRequestContext is the internal diff context consumed by later review
// phases without exposing GitLab response types.
type MergeRequestContext struct {
	Project  ProjectIdentity      `json:"project"`
	MR       MergeRequestMetadata `json:"mr"`
	BaseSHA  string               `json:"base_sha"`
	StartSHA string               `json:"start_sha"`
	HeadSHA  string               `json:"head_sha"`
	Files    []ChangedFile        `json:"files"`
}

// ChangedFile represents a changed file plus inline-commentable line positions.
type ChangedFile struct {
	OldPath   string         `json:"old_path"`
	NewPath   string         `json:"new_path"`
	Deleted   bool           `json:"deleted"`
	Renamed   bool           `json:"renamed"`
	NewFile   bool           `json:"new_file"`
	Positions []DiffPosition `json:"positions"`
}

// DiffPosition contains the data GitLab needs later for inline text positions.
// Phase 3 maps added and changed lines on the new side of the diff.
type DiffPosition struct {
	BaseSHA      string `json:"base_sha"`
	StartSHA     string `json:"start_sha"`
	HeadSHA      string `json:"head_sha"`
	OldPath      string `json:"old_path"`
	NewPath      string `json:"new_path"`
	OldLine      *int   `json:"old_line,omitempty"`
	NewLine      int    `json:"new_line"`
	PositionType string `json:"position_type"`
	Kind         string `json:"kind"`
}
