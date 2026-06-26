package reviews

import (
	"context"
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
	"co-review/server/internal/skills"
)

type Service struct {
	Repo     *Repository
	Platform platform.PlatformClient
	Provider provider.ModelProvider
	Skills   []skills.Skill
	Broker   *events.Broker
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Review, error) {
	if req.MRIID <= 0 {
		return Review{}, errInvalidInput("mr_iid must be positive")
	}
	if strings.TrimSpace(req.ProjectURL) == "" && strings.TrimSpace(req.ProjectPath) == "" {
		return Review{}, errInvalidInput("project_url or project_path is required")
	}
	if s.Repo == nil || s.Platform == nil || s.Provider == nil {
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
	repoID, err := s.Repo.UpsertRepo(ctx, project.Path, project.WebURL, project.Platform)
	if err != nil {
		return Review{}, err
	}
	review := Review{ID: stableID("review", fmt.Sprintf("%s:%d:%d", project.Path, req.MRIID, time.Now().UnixNano())), RepoID: repoID, ProjectPath: project.Path, ProjectURL: project.WebURL, Platform: project.Platform, MRID: strconv.Itoa(req.MRIID), Status: StatusPending, ModelUsed: s.Provider.Name()}
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
	results := s.runHarnesses(ctx, review.ID, mrCtx)
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

func (s *Service) runHarnesses(ctx context.Context, reviewID string, mrCtx platform.MergeRequestContext) []harness.Result {
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
			result := harness.Run(ctx, harness.Config{Dimension: sk.Dimension, Timeout: time.Duration(sk.Harness.TimeoutSeconds) * time.Second, MaxRetries: sk.Harness.MaxRetries, OutputSchema: sk.Harness.OutputSchema, MaxTokens: 1200}, s.Provider, harness.AgentPrompt{System: sk.Body, User: buildPrompt(sk.Dimension, mrCtx)})
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

func buildPrompt(dimension string, mrCtx platform.MergeRequestContext) string {
	data, _ := json.Marshal(mrCtx)
	return fmt.Sprintf("Review dimension: %s\nMerge request context JSON:\n%s", dimension, data)
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
