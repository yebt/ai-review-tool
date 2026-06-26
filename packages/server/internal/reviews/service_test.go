package reviews

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"co-review/server/internal/db"
	"co-review/server/internal/events"
	"co-review/server/internal/harness"
	"co-review/server/internal/platform"
	"co-review/server/internal/provider"
)

func TestServiceCreatePersistsGeneratedReviewAndPendingComments(t *testing.T) {
	database := migratedDB(t)
	broker := events.NewBroker()
	service := &Service{Repo: NewRepository(database), Platform: fakePlatform{}, Provider: provider.DeterministicReviewProvider{}, Broker: broker}

	review, err := service.Create(context.Background(), CreateRequest{ProjectURL: "https://gitlab.com/acme/widget", MRIID: 7})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if review.Status != StatusAwaitingApproval {
		t.Fatalf("status = %q, want %q", review.Status, StatusAwaitingApproval)
	}
	if review.Verdict != "pass" {
		t.Fatalf("verdict = %q, want pass", review.Verdict)
	}
	if len(review.Scores) == 0 {
		t.Fatalf("scores were not persisted")
	}
	comments, err := service.Comments(context.Background(), review.ID)
	if err != nil {
		t.Fatalf("Comments() error = %v", err)
	}
	if len(comments) != 4 {
		t.Fatalf("comments count = %d, want 4", len(comments))
	}
	for _, comment := range comments {
		if comment.Status != CommentStatusPending {
			t.Fatalf("comment %s status = %q, want pending", comment.ID, comment.Status)
		}
	}

	stored, err := service.Get(context.Background(), review.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Status != StatusAwaitingApproval || stored.MRTitle != "Improve widgets" || stored.HeadSHA != "head-sha" {
		t.Fatalf("stored review mismatch: %+v", stored)
	}
}

func TestServiceCreateMarksCleanNoCommentReviewGenerated(t *testing.T) {
	database := migratedDB(t)
	service := &Service{Repo: NewRepository(database), Platform: fakePlatform{}, Provider: noFindingsProvider{}, Broker: events.NewBroker()}

	review, err := service.Create(context.Background(), CreateRequest{ProjectURL: "https://gitlab.com/acme/widget", MRIID: 7})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if review.Status != StatusGenerated {
		t.Fatalf("status = %q, want %q", review.Status, StatusGenerated)
	}
	comments, err := service.Comments(context.Background(), review.ID)
	if err != nil {
		t.Fatalf("Comments() error = %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments count = %d, want 0", len(comments))
	}
}

func TestServiceCreateStoresErrorWhenProviderFails(t *testing.T) {
	database := migratedDB(t)
	service := &Service{Repo: NewRepository(database), Platform: fakePlatform{}, Provider: failingProvider{}, Broker: events.NewBroker()}

	review, err := service.Create(context.Background(), CreateRequest{ProjectURL: "https://gitlab.com/acme/widget", MRIID: 7})
	if err == nil {
		t.Fatalf("Create() error = nil, want provider failure")
	}
	stored, getErr := service.Get(context.Background(), review.ID)
	if getErr != nil {
		t.Fatalf("Get() error = %v", getErr)
	}
	if stored.Status != StatusError {
		t.Fatalf("status = %q, want error", stored.Status)
	}
	if len(stored.Error) == 0 {
		t.Fatalf("expected structured review error")
	}
}

func TestServiceCreateValidatesInput(t *testing.T) {
	database := migratedDB(t)
	service := &Service{Repo: NewRepository(database), Platform: fakePlatform{}, Provider: provider.DeterministicReviewProvider{}}

	_, err := service.Create(context.Background(), CreateRequest{ProjectURL: "https://gitlab.com/acme/widget", MRIID: 0})
	if !IsInvalidInput(err) {
		t.Fatalf("Create() error = %v, want invalid input", err)
	}
}

func TestServiceCreatePersistsPlatformFailure(t *testing.T) {
	database := migratedDB(t)
	service := &Service{Repo: NewRepository(database), Platform: fakePlatform{fetchErr: errors.New("gitlab unavailable")}, Provider: provider.DeterministicReviewProvider{}}

	review, err := service.Create(context.Background(), CreateRequest{ProjectURL: "https://gitlab.com/acme/widget", MRIID: 7})
	if err == nil {
		t.Fatalf("Create() error = nil, want platform failure")
	}
	stored, getErr := service.Get(context.Background(), review.ID)
	if getErr != nil {
		t.Fatalf("Get() error = %v", getErr)
	}
	if stored.Status != StatusError || len(stored.Error) == 0 {
		t.Fatalf("stored failure mismatch: %+v", stored)
	}
}

func TestServiceCreatePersistsErrorOnDatabaseWriteFailures(t *testing.T) {
	cases := []struct {
		name string
		op   string
		code string
	}{
		{name: "running state", op: "update_review_state:running", code: "REVIEW_STATE_UPDATE_FAILED"},
		{name: "review context", op: "update_review_context", code: "REVIEW_CONTEXT_UPDATE_FAILED"},
		{name: "comments insert", op: "insert_comments", code: "REVIEW_COMMENTS_INSERT_FAILED"},
		{name: "final state", op: "update_review_state:awaiting_approval", code: "REVIEW_FINAL_STATE_UPDATE_FAILED"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			database := migratedDB(t)
			repo := NewRepository(database)
			repo.failOnce = map[string]error{tt.op: errors.New("forced db failure")}
			service := &Service{Repo: repo, Platform: fakePlatform{}, Provider: provider.DeterministicReviewProvider{}, Broker: events.NewBroker()}

			review, err := service.Create(context.Background(), CreateRequest{ProjectURL: "https://gitlab.com/acme/widget", MRIID: 7})
			if err == nil {
				t.Fatalf("Create() error = nil, want persistence failure")
			}
			stored, getErr := service.Get(context.Background(), review.ID)
			if getErr != nil {
				t.Fatalf("Get() error = %v", getErr)
			}
			if stored.Status != StatusError {
				t.Fatalf("status = %q, want error", stored.Status)
			}
			if !jsonContains(stored.Error, tt.code) {
				t.Fatalf("error payload = %s, want code %s", stored.Error, tt.code)
			}
		})
	}
}

func TestBuildOutputsAggregatesVerdictPrecedence(t *testing.T) {
	cases := []struct {
		name     string
		outputs  []string
		want     string
		wantCode string
	}{
		{name: "all pass", outputs: []string{agentOutput("risk", 90, "pass"), agentOutput("readability", 88, "pass")}, want: "pass"},
		{name: "needs changes beats pass", outputs: []string{agentOutput("risk", 90, "pass"), agentOutput("readability", 72, "needs_changes")}, want: "needs_changes"},
		{name: "block beats needs changes", outputs: []string{agentOutput("risk", 70, "needs_changes"), agentOutput("resilience", 40, "block")}, want: "block"},
		{name: "invalid output reports parse error", outputs: []string{`{`}, want: "pass", wantCode: "OUTPUT_PARSE_ERROR"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			results := make([]harness.Result, 0, len(tt.outputs))
			for _, output := range tt.outputs {
				results = append(results, harness.Result{Dimension: "risk", Output: json.RawMessage(output)})
			}
			_, scores, verdict, errs := buildOutputs("review_1", results)
			if verdict != tt.want {
				t.Fatalf("verdict = %q, want %q", verdict, tt.want)
			}
			if tt.wantCode == "" && len(scores) == 0 {
				t.Fatalf("scores were empty")
			}
			if tt.wantCode != "" {
				if len(errs) != 1 || errs[0].Code != tt.wantCode {
					t.Fatalf("errs = %+v, want code %s", errs, tt.wantCode)
				}
			}
		})
	}
}

type fakePlatform struct{ fetchErr error }

func (f fakePlatform) InferProject(rawURL string) (platform.ProjectIdentity, error) {
	return platform.ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: "acme/widget", WebURL: rawURL}, nil
}

func (f fakePlatform) FetchMergeRequestContext(ctx context.Context, projectPath string, mrIID int) (platform.MergeRequestContext, error) {
	if f.fetchErr != nil {
		return platform.MergeRequestContext{}, f.fetchErr
	}
	return platform.MergeRequestContext{
		Project: platform.ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: projectPath, WebURL: "https://gitlab.com/" + projectPath},
		MR:      platform.MergeRequestMetadata{IID: mrIID, Title: "Improve widgets", WebURL: "https://gitlab.com/acme/widget/-/merge_requests/7"},
		BaseSHA: "base-sha", StartSHA: "start-sha", HeadSHA: "head-sha",
		Files: []platform.ChangedFile{{NewPath: "README.md", Positions: []platform.DiffPosition{{NewLine: 1}}}},
	}, nil
}

type failingProvider struct{}

func (f failingProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	return provider.CompletionResponse{}, errors.New("provider down")
}
func (f failingProvider) Name() string                          { return "failing" }
func (f failingProvider) SupportedModels() []provider.ModelInfo { return nil }

type noFindingsProvider struct{}

func (p noFindingsProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	dimension := "risk"
	for _, candidate := range []string{"risk", "readability", "reliability", "resilience"} {
		if strings.Contains(strings.ToLower(req.System+"\n"+req.User), candidate) {
			dimension = candidate
			break
		}
	}
	content := agentOutput(dimension, 100, "pass")
	return provider.CompletionResponse{Content: content, ModelUsed: "no-findings"}, nil
}
func (p noFindingsProvider) Name() string                          { return "no-findings" }
func (p noFindingsProvider) SupportedModels() []provider.ModelInfo { return nil }

func jsonContains(raw json.RawMessage, value string) bool {
	return strings.Contains(string(raw), value)
}

func agentOutput(dimension string, score int, verdict string) string {
	data, _ := json.Marshal(map[string]any{"dimension": dimension, "score": score, "findings": []any{}, "summary": "done", "verdict": verdict})
	return string(data)
}

func migratedDB(t *testing.T) *sql.DB {
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
