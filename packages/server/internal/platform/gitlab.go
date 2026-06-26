package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultGitLabBaseURL = "https://gitlab.com"

// GitLabClient fetches merge request context through GitLab's REST API.
type GitLabClient struct {
	baseURL    *url.URL
	token      string
	tokenEnv   string
	httpClient *http.Client
}

// GitLabConfig configures a GitLab REST client.
// Prefer TokenEnv so configuration stores an environment variable reference,
// not a raw secret. Token remains available for focused tests and local wiring
// that already keeps the secret outside persisted configuration.
type GitLabConfig struct {
	BaseURL    string
	Token      string
	TokenEnv   string
	HTTPClient *http.Client
}

func NewGitLabClient(cfg GitLabConfig) (*GitLabClient, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = defaultGitLabBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, platformError(ErrorInvalidProjectURL, "GitLab base URL must be an absolute URL", 0)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	token, err := resolveGitLabToken(cfg)
	if err != nil {
		return nil, err
	}
	return &GitLabClient{baseURL: parsed, token: token, tokenEnv: strings.TrimSpace(cfg.TokenEnv), httpClient: client}, nil
}

func resolveGitLabToken(cfg GitLabConfig) (string, error) {
	envName := strings.TrimSpace(cfg.TokenEnv)
	if envName == "" {
		return strings.TrimSpace(cfg.Token), nil
	}
	if strings.ContainsAny(envName, "= \t\n\r") {
		return "", platformError(ErrorInvalidProjectURL, "GitLab token environment reference must be a variable name", 0)
	}
	return strings.TrimSpace(os.Getenv(envName)), nil
}

func (c *GitLabClient) InferProject(rawURL string) (ProjectIdentity, error) {
	return InferGitLabProject(rawURL)
}

// InferGitLabProject parses HTTPS and SSH-like GitLab project URLs.
func InferGitLabProject(rawURL string) (ProjectIdentity, error) {
	input := strings.TrimSpace(rawURL)
	if input == "" {
		return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "project URL is required", 0)
	}
	if strings.HasPrefix(input, "ssh://") {
		return inferGitLabSSHURLProject(input)
	}
	if strings.Contains(input, "://") {
		return inferGitLabHTTPProject(input)
	}
	if strings.Contains(input, "@") && strings.Contains(input, ":") {
		return inferGitLabSSHProject(input)
	}
	return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "project URL must be HTTPS or SSH-like GitLab URL", 0)
}

func inferGitLabSSHURLProject(input string) (ProjectIdentity, error) {
	parsed, err := url.Parse(input)
	if err != nil || parsed.Scheme != "ssh" || parsed.Host == "" {
		return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "SSH URL must include a host", 0)
	}
	path, err := cleanProjectPath(parsed.Path)
	if err != nil {
		return ProjectIdentity{}, err
	}
	web := url.URL{Scheme: "https", Host: parsed.Host, Path: "/" + path}
	return ProjectIdentity{Platform: "gitlab", Host: parsed.Host, Path: path, WebURL: web.String()}, nil
}

func inferGitLabHTTPProject(input string) (ProjectIdentity, error) {
	parsed, err := url.Parse(input)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "project URL must be absolute", 0)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "project URL must use http or https", 0)
	}
	path, err := cleanProjectPath(parsed.Path)
	if err != nil {
		return ProjectIdentity{}, err
	}
	web := url.URL{Scheme: parsed.Scheme, Host: parsed.Host, Path: "/" + path}
	return ProjectIdentity{Platform: "gitlab", Host: parsed.Host, Path: path, WebURL: web.String()}, nil
}

func inferGitLabSSHProject(input string) (ProjectIdentity, error) {
	parts := strings.SplitN(input, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "SSH-like URL must include a user and host", 0)
	}
	hostAndPath := strings.SplitN(parts[1], ":", 2)
	if len(hostAndPath) != 2 || hostAndPath[0] == "" {
		return ProjectIdentity{}, platformError(ErrorInvalidProjectURL, "SSH-like URL must include a host and path", 0)
	}
	path, err := cleanProjectPath(hostAndPath[1])
	if err != nil {
		return ProjectIdentity{}, err
	}
	web := url.URL{Scheme: "https", Host: hostAndPath[0], Path: "/" + path}
	return ProjectIdentity{Platform: "gitlab", Host: hostAndPath[0], Path: path, WebURL: web.String()}, nil
}

func cleanProjectPath(rawPath string) (string, error) {
	path := strings.Trim(strings.TrimSpace(rawPath), "/")
	path = strings.TrimSuffix(path, ".git")
	if path == "" || !strings.Contains(path, "/") {
		return "", platformError(ErrorInvalidProjectURL, "project path must include namespace and project", 0)
	}
	if strings.Contains(path, "//") || strings.Contains(path, " ") {
		return "", platformError(ErrorInvalidProjectURL, "project path is malformed", 0)
	}
	return path, nil
}

func (c *GitLabClient) FetchMergeRequestContext(ctx context.Context, projectPath string, mrIID int) (MergeRequestContext, error) {
	if strings.TrimSpace(projectPath) == "" {
		return MergeRequestContext{}, platformError(ErrorInvalidProjectURL, "project path is required", 0)
	}
	if mrIID <= 0 {
		return MergeRequestContext{}, platformError(ErrorInvalidMR, "merge request IID must be positive", 0)
	}

	var mr gitLabMRResponse
	if err := c.getJSON(ctx, c.projectAPIPath(projectPath, "merge_requests", strconv.Itoa(mrIID)), &mr); err != nil {
		return MergeRequestContext{}, err
	}

	var changes gitLabChangesResponse
	if err := c.getJSON(ctx, c.projectAPIPath(projectPath, "merge_requests", strconv.Itoa(mrIID), "changes"), &changes); err != nil {
		return MergeRequestContext{}, err
	}

	ctxModel := MergeRequestContext{
		Project:  ProjectIdentity{Platform: "gitlab", Host: c.baseURL.Host, Path: projectPath, WebURL: c.projectWebURL(projectPath)},
		MR:       MergeRequestMetadata{IID: mr.IID, Title: mr.Title, WebURL: mr.WebURL},
		BaseSHA:  changes.DiffRefs.BaseSHA,
		StartSHA: changes.DiffRefs.StartSHA,
		HeadSHA:  changes.DiffRefs.HeadSHA,
	}
	if ctxModel.MR.IID == 0 {
		ctxModel.MR.IID = mrIID
	}
	if ctxModel.MR.Title == "" {
		ctxModel.MR.Title = changes.Title
	}
	if ctxModel.MR.WebURL == "" {
		ctxModel.MR.WebURL = changes.WebURL
	}
	if ctxModel.BaseSHA == "" || ctxModel.StartSHA == "" || ctxModel.HeadSHA == "" {
		return MergeRequestContext{}, platformError(ErrorMalformedResponse, "GitLab changes response is missing diff refs", 0)
	}

	for _, change := range changes.Changes {
		file := ChangedFile{OldPath: change.OldPath, NewPath: change.NewPath, Deleted: change.DeletedFile, Renamed: change.RenamedFile, NewFile: change.NewFile}
		positions, err := MapDiffPositions(change.Diff, file, ctxModel.BaseSHA, ctxModel.StartSHA, ctxModel.HeadSHA)
		if err != nil {
			return MergeRequestContext{}, platformError(ErrorMalformedResponse, err.Error(), 0)
		}
		file.Positions = positions
		ctxModel.Files = append(ctxModel.Files, file)
	}

	return ctxModel, nil
}

func (c *GitLabClient) projectAPIPath(projectPath string, segments ...string) string {
	pathSegments := append([]string{"api", "v4", "projects", url.PathEscape(projectPath)}, segments...)
	return "/" + strings.Join(pathSegments, "/")
}

func (c *GitLabClient) projectWebURL(projectPath string) string {
	web := *c.baseURL
	web.Path = "/" + strings.Trim(projectPath, "/")
	web.RawQuery = ""
	web.Fragment = ""
	return web.String()
}

func (c *GitLabClient) getJSON(ctx context.Context, apiPath string, target any) error {
	endpoint := strings.TrimRight(c.baseURL.String(), "/") + apiPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.gitLabStatusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return platformError(ErrorMalformedResponse, err.Error(), resp.StatusCode)
	}
	return nil
}

func (c *GitLabClient) gitLabStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		if c.token == "" {
			envName := c.tokenEnv
			if envName == "" {
				envName = "CO_REVIEW_GITLAB_TOKEN"
			}
			message = fmt.Sprintf("GitLab API returned %s and no token is configured; set %s for private merge requests: %s", http.StatusText(resp.StatusCode), envName, message)
		}
		return platformError(ErrorUnauthorized, message, resp.StatusCode)
	case http.StatusNotFound:
		return platformError(ErrorNotFound, message, resp.StatusCode)
	default:
		return platformError(ErrorHTTP, fmt.Sprintf("GitLab API request failed: %s", message), resp.StatusCode)
	}
}

type gitLabMRResponse struct {
	IID    int    `json:"iid"`
	Title  string `json:"title"`
	WebURL string `json:"web_url"`
}

type gitLabChangesResponse struct {
	IID      int                   `json:"iid"`
	Title    string                `json:"title"`
	WebURL   string                `json:"web_url"`
	DiffRefs gitLabDiffRefs        `json:"diff_refs"`
	Changes  []gitLabChangePayload `json:"changes"`
}

type gitLabDiffRefs struct {
	BaseSHA  string `json:"base_sha"`
	StartSHA string `json:"start_sha"`
	HeadSHA  string `json:"head_sha"`
}

type gitLabChangePayload struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}
