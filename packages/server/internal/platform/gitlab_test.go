package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInferGitLabProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ProjectIdentity
		wantErr bool
	}{
		{
			name:  "https simple",
			input: "https://gitlab.com/group/project",
			want:  ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: "group/project", WebURL: "https://gitlab.com/group/project"},
		},
		{
			name:  "https nested namespace with git suffix",
			input: "https://gitlab.com/group/subgroup/project.git",
			want:  ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: "group/subgroup/project", WebURL: "https://gitlab.com/group/subgroup/project"},
		},
		{
			name:  "https trailing slash",
			input: "https://gitlab.example.com/team/project.git/",
			want:  ProjectIdentity{Platform: "gitlab", Host: "gitlab.example.com", Path: "team/project", WebURL: "https://gitlab.example.com/team/project"},
		},
		{
			name:  "ssh like",
			input: "git@gitlab.com:group/project.git",
			want:  ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: "group/project", WebURL: "https://gitlab.com/group/project"},
		},
		{
			name:  "ssh url",
			input: "ssh://git@gitlab.example.com/group/subgroup/project.git",
			want:  ProjectIdentity{Platform: "gitlab", Host: "gitlab.example.com", Path: "group/subgroup/project", WebURL: "https://gitlab.example.com/group/subgroup/project"},
		},
		{name: "empty input", input: " ", wantErr: true},
		{name: "missing namespace", input: "https://gitlab.com/project", wantErr: true},
		{name: "unsupported scheme", input: "ftp://gitlab.com/group/project", wantErr: true},
		{name: "malformed path", input: "https://gitlab.com/group//project", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferGitLabProject(tt.input)
			if tt.wantErr {
				assertPlatformErrorCode(t, err, ErrorInvalidProjectURL)
				return
			}
			if err != nil {
				t.Fatalf("InferGitLabProject() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("InferGitLabProject() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGitLabFetchMergeRequestContextHappyPath(t *testing.T) {
	t.Parallel()

	server := newFakeGitLabServer(t, fakeGitLabConfig{expectedToken: "test-token"})
	client := newTestGitLabClient(t, GitLabConfig{BaseURL: server.URL, Token: "test-token", HTTPClient: server.HTTPClient})

	got, err := client.FetchMergeRequestContext(context.Background(), "group/project", 7)
	if err != nil {
		t.Fatalf("FetchMergeRequestContext() unexpected error: %v", err)
	}
	if got.Project.Path != "group/project" || got.Project.Host == "" {
		t.Fatalf("project = %+v, want group/project on fake host", got.Project)
	}
	if got.MR.IID != 7 || got.MR.Title != "Improve parser" || got.MR.WebURL != "https://gitlab.example.com/group/project/-/merge_requests/7" {
		t.Fatalf("MR = %+v, want fetched metadata", got.MR)
	}
	if got.BaseSHA != "base-sha" || got.StartSHA != "start-sha" || got.HeadSHA != "head-sha" {
		t.Fatalf("diff refs = %q/%q/%q, want base/start/head", got.BaseSHA, got.StartSHA, got.HeadSHA)
	}
	if len(got.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(got.Files))
	}
	if got.Files[0].OldPath != "old.go" || got.Files[0].NewPath != "new.go" || !got.Files[0].Renamed {
		t.Fatalf("first file = %+v, want renamed old.go -> new.go", got.Files[0])
	}
	if len(got.Files[0].Positions) != 2 {
		t.Fatalf("first file positions = %d, want 2", len(got.Files[0].Positions))
	}
	assertPosition(t, got.Files[0].Positions[0], "changed", 10, "old.go", "new.go")
	assertPosition(t, got.Files[0].Positions[1], "added", 12, "old.go", "new.go")
	if len(got.Files[1].Positions) != 1 {
		t.Fatalf("second file positions = %d, want 1", len(got.Files[1].Positions))
	}
	assertPosition(t, got.Files[1].Positions[0], "added", 1, "created.go", "created.go")
	if !server.sawToken {
		t.Fatal("fake GitLab server did not receive PRIVATE-TOKEN header")
	}
}

func TestGitLabClientUsesTokenEnvReference(t *testing.T) {
	server := newFakeGitLabServer(t, fakeGitLabConfig{expectedToken: "env-token"})
	t.Setenv("CO_REVIEW_GITLAB_TOKEN_TEST", "env-token")
	client := newTestGitLabClient(t, GitLabConfig{BaseURL: server.URL, TokenEnv: "CO_REVIEW_GITLAB_TOKEN_TEST", Token: "raw-token", HTTPClient: server.HTTPClient})

	_, err := client.FetchMergeRequestContext(context.Background(), "group/project", 7)
	if err != nil {
		t.Fatalf("FetchMergeRequestContext() unexpected error: %v", err)
	}
	if !server.sawToken {
		t.Fatal("fake GitLab server did not receive token from environment reference")
	}
}

func TestNewGitLabClientRejectsTokenEnvValueInsteadOfReference(t *testing.T) {
	t.Parallel()

	_, err := NewGitLabClient(GitLabConfig{BaseURL: "https://gitlab.example.com", TokenEnv: "CO_REVIEW_GITLAB_TOKEN=secret"})
	assertPlatformErrorCode(t, err, ErrorInvalidProjectURL)
}

func TestGitLabFetchMergeRequestContextClarifiesMissingTokenOnUnauthorized(t *testing.T) {
	server := newFakeGitLabServer(t, fakeGitLabConfig{metadataStatus: http.StatusUnauthorized})
	client := newTestGitLabClient(t, GitLabConfig{BaseURL: server.URL, TokenEnv: "CO_REVIEW_GITLAB_TOKEN_TEST", HTTPClient: server.HTTPClient})

	_, err := client.FetchMergeRequestContext(context.Background(), "group/project", 7)
	assertPlatformErrorCode(t, err, ErrorUnauthorized)
	if !strings.Contains(err.Error(), "no token is configured") || !strings.Contains(err.Error(), "CO_REVIEW_GITLAB_TOKEN_TEST") {
		t.Fatalf("error = %v, want missing-token guidance", err)
	}
}

func TestGitLabFetchMergeRequestContextErrorResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      fakeGitLabConfig
		wantCode string
	}{
		{name: "metadata not found", cfg: fakeGitLabConfig{metadataStatus: http.StatusNotFound}, wantCode: ErrorNotFound},
		{name: "metadata forbidden", cfg: fakeGitLabConfig{metadataStatus: http.StatusForbidden}, wantCode: ErrorUnauthorized},
		{name: "metadata server error", cfg: fakeGitLabConfig{metadataStatus: http.StatusInternalServerError}, wantCode: ErrorHTTP},
		{name: "changes unauthorized", cfg: fakeGitLabConfig{changesStatus: http.StatusUnauthorized}, wantCode: ErrorUnauthorized},
		{name: "metadata malformed", cfg: fakeGitLabConfig{metadataBody: `{not-json`}, wantCode: ErrorMalformedResponse},
		{name: "changes malformed", cfg: fakeGitLabConfig{changesBody: `{not-json`}, wantCode: ErrorMalformedResponse},
		{name: "missing diff refs", cfg: fakeGitLabConfig{changesBody: `{"changes":[]}`}, wantCode: ErrorMalformedResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := newFakeGitLabServer(t, tt.cfg)
			client := newTestGitLabClient(t, GitLabConfig{BaseURL: server.URL, HTTPClient: server.HTTPClient})

			_, err := client.FetchMergeRequestContext(context.Background(), "group/project", 7)
			assertPlatformErrorCode(t, err, tt.wantCode)
		})
	}
}

func TestGitLabFetchMergeRequestContextRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	client := newTestGitLabClient(t, GitLabConfig{BaseURL: "https://gitlab.example.com"})

	_, err := client.FetchMergeRequestContext(context.Background(), "", 7)
	assertPlatformErrorCode(t, err, ErrorInvalidProjectURL)

	_, err = client.FetchMergeRequestContext(context.Background(), "group/project", 0)
	assertPlatformErrorCode(t, err, ErrorInvalidMR)
}

func TestGitLabFetchMergeRequestContextTransportFailure(t *testing.T) {
	t.Parallel()

	client := newTestGitLabClient(t, GitLabConfig{
		BaseURL: "https://gitlab.example.com",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("transport unavailable")
		})},
	})

	_, err := client.FetchMergeRequestContext(context.Background(), "group/project", 7)
	if err == nil || !strings.Contains(err.Error(), "transport unavailable") {
		t.Fatalf("FetchMergeRequestContext() error = %v, want transport failure", err)
	}
}

func TestMapDiffPositionsAddedAndChangedLines(t *testing.T) {
	t.Parallel()

	file := ChangedFile{OldPath: "app.go", NewPath: "app.go"}
	diff := strings.Join([]string{
		"@@ -3,5 +3,6 @@",
		" context",
		"-old value",
		"+new value",
		" unchanged",
		"+added value",
		" another context",
	}, "\n")

	got, err := MapDiffPositions(diff, file, "base", "start", "head")
	if err != nil {
		t.Fatalf("MapDiffPositions() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("positions = %d, want 2", len(got))
	}
	assertPosition(t, got[0], "changed", 4, "app.go", "app.go")
	assertPosition(t, got[1], "added", 6, "app.go", "app.go")
	for _, pos := range got {
		if pos.BaseSHA != "base" || pos.StartSHA != "start" || pos.HeadSHA != "head" || pos.PositionType != "text" {
			t.Fatalf("position refs/type = %+v, want base/start/head text", pos)
		}
	}
}

func TestMapDiffPositionsRejectsMalformedHunk(t *testing.T) {
	t.Parallel()

	_, err := MapDiffPositions("@@ malformed @@\n+line", ChangedFile{OldPath: "a", NewPath: "a"}, "base", "start", "head")
	if err == nil {
		t.Fatal("MapDiffPositions() error = nil, want malformed hunk error")
	}
}

func TestMapDiffPositionsMultipleHunks(t *testing.T) {
	t.Parallel()

	diff := strings.Join([]string{
		"@@ -1,2 +1,2 @@",
		"-old first",
		"+new first",
		" context",
		"@@ -20,2 +20,3 @@",
		" context",
		"+second add",
	}, "\n")

	got, err := MapDiffPositions(diff, ChangedFile{OldPath: "app.go", NewPath: "app.go"}, "base", "start", "head")
	if err != nil {
		t.Fatalf("MapDiffPositions() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("positions = %d, want 2", len(got))
	}
	assertPosition(t, got[0], "changed", 1, "app.go", "app.go")
	assertPosition(t, got[1], "added", 21, "app.go", "app.go")
}

func TestMapDiffPositionsPureAdditionHunk(t *testing.T) {
	t.Parallel()

	diff := strings.Join([]string{
		"@@ -0,0 +1,2 @@",
		"+package created",
		"+func init() {}",
	}, "\n")

	got, err := MapDiffPositions(diff, ChangedFile{OldPath: "created.go", NewPath: "created.go", NewFile: true}, "base", "start", "head")
	if err != nil {
		t.Fatalf("MapDiffPositions() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("positions = %d, want 2", len(got))
	}
	assertPosition(t, got[0], "added", 1, "created.go", "created.go")
	assertPosition(t, got[1], "added", 2, "created.go", "created.go")
}

func TestMapDiffPositionsEmptyAndBinaryDiffs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		diff string
	}{
		{name: "empty", diff: ""},
		{name: "binary", diff: "Binary files a/image.png and b/image.png differ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MapDiffPositions(tt.diff, ChangedFile{OldPath: "image.png", NewPath: "image.png"}, "base", "start", "head")
			if err != nil {
				t.Fatalf("MapDiffPositions() unexpected error: %v", err)
			}
			if len(got) != 0 {
				t.Fatalf("positions = %d, want 0", len(got))
			}
		})
	}
}

type fakeGitLabConfig struct {
	metadataStatus int
	changesStatus  int
	metadataBody   string
	changesBody    string
	expectedToken  string
}

type fakeGitLabServer struct {
	URL        string
	sawToken   bool
	HTTPClient *http.Client
}

func newFakeGitLabServer(t *testing.T, cfg fakeGitLabConfig) *fakeGitLabServer {
	t.Helper()
	fake := &fakeGitLabServer{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.expectedToken != "" && r.Header.Get("PRIVATE-TOKEN") == cfg.expectedToken {
			fake.sawToken = true
		}
		wantPrefix := "/api/v4/projects/group%2Fproject/merge_requests/7"
		if r.URL.EscapedPath() != wantPrefix && r.URL.EscapedPath() != wantPrefix+"/changes" {
			t.Fatalf("request path = %q, want GitLab project MR path", r.URL.EscapedPath())
		}

		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.EscapedPath(), "/changes") {
			status := cfg.changesStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			body := cfg.changesBody
			if body == "" {
				body = defaultChangesBody(t)
			}
			_, _ = w.Write([]byte(body))
			return
		}

		status := cfg.metadataStatus
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		body := cfg.metadataBody
		if body == "" {
			body = `{"iid":7,"title":"Improve parser","web_url":"https://gitlab.example.com/group/project/-/merge_requests/7"}`
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	fake.URL = server.URL
	fake.HTTPClient = server.Client()
	return fake
}

func defaultChangesBody(t *testing.T) string {
	t.Helper()
	payload := map[string]any{
		"diff_refs": map[string]string{"base_sha": "base-sha", "start_sha": "start-sha", "head_sha": "head-sha"},
		"changes": []map[string]any{
			{
				"old_path":     "old.go",
				"new_path":     "new.go",
				"renamed_file": true,
				"diff":         "@@ -10,3 +10,4 @@\n-old line\n+new line\n context\n+added line\n",
			},
			{
				"old_path": "created.go",
				"new_path": "created.go",
				"new_file": true,
				"diff":     "@@ -0,0 +1,1 @@\n+package created\n",
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal changes fixture: %v", err)
	}
	return string(body)
}

func newTestGitLabClient(t *testing.T, cfg GitLabConfig) *GitLabClient {
	t.Helper()
	client, err := NewGitLabClient(cfg)
	if err != nil {
		t.Fatalf("NewGitLabClient() error: %v", err)
	}
	return client
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func assertPlatformErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want platform error code %s", wantCode)
	}
	var platformErr *Error
	if !errors.As(err, &platformErr) {
		t.Fatalf("error = %T %[1]v, want *platform.Error", err)
	}
	if platformErr.Code != wantCode {
		t.Fatalf("error code = %s, want %s", platformErr.Code, wantCode)
	}
}

func assertPosition(t *testing.T, got DiffPosition, wantKind string, wantNewLine int, wantOldPath string, wantNewPath string) {
	t.Helper()
	if got.Kind != wantKind || got.NewLine != wantNewLine || got.OldPath != wantOldPath || got.NewPath != wantNewPath {
		t.Fatalf("position = %+v, want kind=%s newLine=%d oldPath=%s newPath=%s", got, wantKind, wantNewLine, wantOldPath, wantNewPath)
	}
}
