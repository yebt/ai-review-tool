package reviews

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"co-review/server/internal/events"
	"co-review/server/internal/harness"
	"co-review/server/internal/platform"
	"co-review/server/internal/provider"
	repomemory "co-review/server/internal/repos"
	"co-review/server/internal/skills"
)

type Service struct {
	Repo     *Repository
	Platform platform.PlatformClient
	Provider provider.ModelProvider
	Skills   []skills.Skill
	Broker   *events.Broker
	Memory   *repomemory.Service
	Repos    *repomemory.Service
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Review, error) {
	if req.MRIID <= 0 {
		return Review{}, errInvalidInput("mr_iid must be positive")
	}
	if strings.TrimSpace(req.ProjectURL) == "" && strings.TrimSpace(req.ProjectPath) == "" {
		return Review{}, errInvalidInput("project_url or project_path is required")
	}
	if s.Repo == nil || s.Platform == nil || s.Provider == nil || s.repoService() == nil {
		return Review{}, errors.New("review service dependencies are not configured")
	}
	project := platform.ProjectIdentity{Platform: "gitlab", Path: strings.TrimSpace(req.ProjectPath), WebURL: strings.TrimSpace(req.ProjectURL)}
	if project.Path == "" {
		inferred, err := s.Platform.InferProject(req.ProjectURL)
		if err != nil {
			return Review{}, err
		}
		project = inferred
	}
	repoID, err := s.ensureRepo(ctx, project)
	if err != nil {
		return Review{}, err
	}
	modelSelection, err := s.resolveModel(ctx, repoID)
	if err != nil {
		return Review{}, err
	}
	review := Review{ID: stableID("review", fmt.Sprintf("%s:%d:%d", project.Path, req.MRIID, time.Now().UnixNano())), RepoID: repoID, ProjectPath: project.Path, ProjectURL: project.WebURL, Platform: project.Platform, MRID: strconv.Itoa(req.MRIID), Status: StatusPending, ModelUsed: modelSelection.ModelUsed()}
	if err := s.Repo.InsertReview(ctx, review); err != nil {
		return Review{}, err
	}
	s.publish(review.ID, "review.started", map[string]any{"review_id": review.ID, "status": StatusPending})
	if err := s.Repo.UpdateReviewState(ctx, review.ID, StatusRunning, nil, "", nil, false); err != nil {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, []HarnessError{{Dimension: "persistence", Code: "REVIEW_STATE_UPDATE_FAILED", Message: err.Error()}}))
	}
	s.publish(review.ID, "review.started", map[string]any{"review_id": review.ID, "status": StatusRunning})
	mrCtx, err := s.Platform.FetchMergeRequestContext(ctx, project.Path, req.MRIID)
	if err != nil {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, []HarnessError{{Dimension: "platform", Code: "PLATFORM_ERROR", Message: err.Error()}}))
	}
	review.MRURL, review.MRTitle, review.BaseSHA, review.StartSHA, review.HeadSHA = mrCtx.MR.WebURL, mrCtx.MR.Title, mrCtx.BaseSHA, mrCtx.StartSHA, mrCtx.HeadSHA
	if err := s.Repo.UpdateReviewContext(ctx, review); err != nil {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, []HarnessError{{Dimension: "persistence", Code: "REVIEW_CONTEXT_UPDATE_FAILED", Message: err.Error()}}))
	}
	memoryContext, err := s.memoryContext(ctx, review.RepoID)
	if err != nil {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, []HarnessError{{Dimension: "memory", Code: "MEMORY_CONTEXT_FAILED", Message: err.Error()}}))
	}
	results := s.runHarnesses(ctx, review.ID, mrCtx, memoryContext, modelSelection)
	comments, scores, verdict, harnessErrors := buildOutputs(review.ID, results)
	if len(harnessErrors) > 0 {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, harnessErrors))
	}
	if err := s.Repo.InsertComments(ctx, comments); err != nil {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, []HarnessError{{Dimension: "persistence", Code: "REVIEW_COMMENTS_INSERT_FAILED", Message: err.Error()}}))
	}
	finalStatus := generatedStatus(comments)
	if err := s.Repo.UpdateReviewState(ctx, review.ID, finalStatus, scores, verdict, nil, true); err != nil {
		return s.reviewAfterFailure(ctx, review, s.fail(ctx, review.ID, []HarnessError{{Dimension: "persistence", Code: "REVIEW_FINAL_STATE_UPDATE_FAILED", Message: err.Error()}}))
	}
	s.publish(review.ID, "review.generated", map[string]any{"review_id": review.ID, "status": finalStatus, "comments": len(comments), "verdict": verdict})
	stored, err := s.Repo.GetReview(ctx, review.ID)
	if err != nil {
		return Review{}, err
	}
	stored.Comments, _ = s.Repo.ListComments(ctx, review.ID)
	return stored, nil
}

func (s *Service) List(ctx context.Context) ([]Review, error) { return s.Repo.ListReviews(ctx) }
func (s *Service) Get(ctx context.Context, id string) (Review, error) {
	rv, err := s.Repo.GetReview(ctx, id)
	if err != nil {
		return rv, err
	}
	rv.Comments, _ = s.Repo.ListComments(ctx, id)
	return rv, nil
}
func (s *Service) Comments(ctx context.Context, id string) ([]Comment, error) {
	return s.Repo.ListComments(ctx, id)
}

func (s *Service) UpdateCommentStatus(ctx context.Context, reviewID, commentID, status string) (Comment, error) {
	if !validCommentStatus(status) {
		return Comment{}, errInvalidInput("status must be approved, accepted_decision, or discarded")
	}
	if status == CommentStatusAcceptedDecision {
		if s.repoService() == nil {
			return Comment{}, errors.New("repo memory service is not configured")
		}
		review, err := s.Repo.GetReview(ctx, reviewID)
		if err != nil {
			return Comment{}, err
		}
		comment, err := s.Repo.GetComment(ctx, reviewID, commentID)
		if err != nil {
			return Comment{}, err
		}
		content := strings.TrimSpace(comment.Why)
		if content == "" {
			content = strings.TrimSpace(comment.Evidence)
		}
		entry, err := repomemory.MemoryFromInput(review.RepoID, repomemory.MemoryInput{Type: repomemory.MemoryTypeAcceptedDecision, Key: comment.ID, Content: content, Dimension: comment.Dimension, SourceMR: review.MRID})
		if err != nil {
			return Comment{}, err
		}
		return s.Repo.AcceptCommentAsDecision(ctx, reviewID, commentID, entry)
	}
	return s.Repo.UpdateCommentStatus(ctx, reviewID, commentID, status)
}

func (s *Service) fail(ctx context.Context, reviewID string, errs []HarnessError) error {
	raw, _ := json.Marshal(map[string]any{"errors": errs})
	_ = s.Repo.UpdateReviewState(ctx, reviewID, StatusError, nil, "", raw, true)
	s.publish(reviewID, "review.error", map[string]any{"review_id": reviewID, "status": StatusError, "errors": errs})
	return fmt.Errorf("review failed")
}

func (s *Service) reviewAfterFailure(ctx context.Context, fallback Review, err error) (Review, error) {
	stored, getErr := s.Repo.GetReview(ctx, fallback.ID)
	if getErr != nil {
		return fallback, err
	}
	stored.Comments, _ = s.Repo.ListComments(ctx, fallback.ID)
	return stored, err
}

func (s *Service) runHarnesses(ctx context.Context, reviewID string, mrCtx platform.MergeRequestContext, memoryContext string, model modelSelection) []harness.Result {
	skillsToRun := s.Skills
	if len(skillsToRun) == 0 {
		skillsToRun = defaultSkills()
	}
	results := make([]harness.Result, len(skillsToRun))
	var wg sync.WaitGroup
	for i, sk := range skillsToRun {
		i, sk := i, sk
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.publish(reviewID, "agent.started", map[string]any{"review_id": reviewID, "dimension": sk.Dimension})
			result := harness.Run(ctx, harness.Config{Dimension: sk.Dimension, Timeout: time.Duration(sk.Harness.TimeoutSeconds) * time.Second, MaxRetries: sk.Harness.MaxRetries, OutputSchema: sk.Harness.OutputSchema, MaxTokens: 1200, ProviderName: model.Provider, ModelName: model.Model}, s.Provider, harness.AgentPrompt{System: sk.Body, User: buildPrompt(sk.Dimension, mrCtx, memoryContext)})
			results[i] = result
			if result.Error != nil {
				s.publish(reviewID, "agent.error", map[string]any{"review_id": reviewID, "dimension": sk.Dimension, "error": result.Error})
				return
			}
			s.publish(reviewID, "agent.completed", map[string]any{"review_id": reviewID, "dimension": sk.Dimension, "attempts": result.Attempts})
		}()
	}
	wg.Wait()
	return results
}

func buildPrompt(dimension string, mrCtx platform.MergeRequestContext, memoryContext string) string {
	data, _ := json.Marshal(mrCtx)
	prompt := fmt.Sprintf("Review dimension: %s\nMerge request context JSON:\n%s", dimension, data)
	if strings.TrimSpace(memoryContext) != "" {
		prompt += "\n\n" + memoryContext
	}
	return prompt
}

func (s *Service) memoryContext(ctx context.Context, repoID string) (string, error) {
	if s.repoService() == nil {
		return "", nil
	}
	return s.repoService().RenderPromptContext(ctx, repoID)
}

func (s *Service) ensureRepo(ctx context.Context, project platform.ProjectIdentity) (string, error) {
	if svc := s.repoService(); svc != nil {
		repo, err := svc.Ensure(ctx, repomemory.RepoInput{Name: project.Path, URL: project.WebURL, Platform: project.Platform})
		if err != nil {
			return "", err
		}
		return repo.ID, nil
	}
	return "", errors.New("repo service is not configured")
}

func (s *Service) repoService() *repomemory.Service {
	if s.Repos != nil {
		return s.Repos
	}
	return s.Memory
}

type modelSelection struct {
	Provider string
	Model    string
	Source   string
}

func (m modelSelection) ModelUsed() string {
	if strings.TrimSpace(m.Provider) == "" {
		return m.Model
	}
	return m.Provider + "/" + m.Model
}

func (s *Service) resolveModel(ctx context.Context, repoID string) (modelSelection, error) {
	if svc := s.repoService(); svc != nil {
		cfg, err := svc.GetModel(ctx, repoID)
		if err == nil {
			return modelSelection{Provider: cfg.Provider, Model: cfg.ModelName, Source: "repo_config"}, nil
		}
		if err != sql.ErrNoRows {
			return modelSelection{}, err
		}
	}
	return modelSelection{Provider: s.Provider.Name(), Model: s.Provider.Name(), Source: "deterministic_fallback"}, nil
}

func validCommentStatus(status string) bool {
	switch status {
	case CommentStatusApproved, CommentStatusAcceptedDecision, CommentStatusDiscarded:
		return true
	default:
		return false
	}
}

func buildOutputs(reviewID string, results []harness.Result) ([]Comment, json.RawMessage, string, []HarnessError) {
	scores := map[string]int{}
	verdict := "pass"
	var comments []Comment
	var errs []HarnessError
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, HarnessError{Dimension: result.Dimension, Code: result.Error.Code, Message: result.Error.Message})
			continue
		}
		var out struct {
			Dimension string `json:"dimension"`
			Score     int    `json:"score"`
			Verdict   string `json:"verdict"`
			Findings  []struct {
				Severity          string `json:"severity"`
				File              string `json:"file"`
				LineStart         int    `json:"line_start"`
				LineEnd           int    `json:"line_end"`
				Evidence          string `json:"evidence"`
				Why               string `json:"why"`
				SuggestionSnippet string `json:"suggestion_snippet"`
				InlineComment     bool   `json:"inline_comment"`
			} `json:"findings"`
		}
		if err := json.Unmarshal(result.Output, &out); err != nil {
			errs = append(errs, HarnessError{Dimension: result.Dimension, Code: "OUTPUT_PARSE_ERROR", Message: err.Error()})
			continue
		}
		scores[out.Dimension] = out.Score
		if out.Verdict == "block" {
			verdict = "block"
		} else if out.Verdict == "needs_changes" && verdict == "pass" {
			verdict = "needs_changes"
		}
		for i, f := range out.Findings {
			ls, le := f.LineStart, f.LineEnd
			comments = append(comments, Comment{ID: stableID("comment", fmt.Sprintf("%s:%s:%d", reviewID, out.Dimension, i)), ReviewID: reviewID, Dimension: out.Dimension, Severity: f.Severity, File: f.File, LineStart: &ls, LineEnd: &le, Evidence: f.Evidence, Why: f.Why, SuggestionSnippet: f.SuggestionSnippet, Status: CommentStatusPending})
		}
	}
	rawScores, _ := json.Marshal(scores)
	return comments, rawScores, verdict, errs
}

func generatedStatus(comments []Comment) string {
	if len(comments) > 0 {
		return StatusAwaitingApproval
	}
	return StatusGenerated
}

func defaultSkills() []skills.Skill {
	dims := []string{"risk", "readability", "reliability", "resilience"}
	out := make([]skills.Skill, 0, len(dims))
	for _, d := range dims {
		out = append(out, skills.Skill{Name: "review-" + d, Dimension: d, Model: "fake", Body: "You are the " + d + " reviewer.", Harness: skills.HarnessConfig{TimeoutSeconds: 5, MaxRetries: 0, OutputSchema: d}})
	}
	return out
}
func (s *Service) publish(id, name string, payload any) {
	if s.Broker != nil {
		s.Broker.Publish(id, name, payload)
	}
}

type invalidInputError struct{ message string }

func (e invalidInputError) Error() string  { return e.message }
func errInvalidInput(message string) error { return invalidInputError{message: message} }
func IsInvalidInput(err error) bool        { var target invalidInputError; return errors.As(err, &target) }
