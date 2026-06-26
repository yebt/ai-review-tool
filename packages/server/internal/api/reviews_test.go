package api

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"co-review/server/internal/db"
	"co-review/server/internal/events"
	"co-review/server/internal/platform"
	"co-review/server/internal/provider"
	"co-review/server/internal/reviews"
)

func TestReviewsAPIEndToEnd(t *testing.T) {
	service, broker := testReviewService(t, provider.DeterministicReviewProvider{})
	router := NewRouterWithDeps(RouterDeps{Reviews: service, Broker: broker})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(`{"project_url":"https://gitlab.com/acme/widget","mr_iid":7}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created struct {
		Review reviews.Review `json:"review"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Review.Status != reviews.StatusAwaitingApproval {
		t.Fatalf("created status = %q, want %q", created.Review.Status, reviews.StatusAwaitingApproval)
	}

	for _, path := range []string{"/api/v1/reviews", "/api/v1/reviews/" + created.Review.ID, "/api/v1/reviews/" + created.Review.ID + "/comments"} {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestReviewsAPIRejectsInvalidMRInput(t *testing.T) {
	service, broker := testReviewService(t, provider.DeterministicReviewProvider{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(`{"project_url":"https://gitlab.com/acme/widget","mr_iid":0}`))

	NewRouterWithDeps(RouterDeps{Reviews: service, Broker: broker}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestReviewsAPIReportsProviderFailure(t *testing.T) {
	service, broker := testReviewService(t, failingAPIProvider{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(`{"project_url":"https://gitlab.com/acme/widget","mr_iid":7}`))

	NewRouterWithDeps(RouterDeps{Reviews: service, Broker: broker}).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", rec.Code, rec.Body.String())
	}
}

func TestReviewsAPIReportsPlatformFailure(t *testing.T) {
	service, broker := testReviewServiceWithPlatform(t, failingAPIPlatform{}, provider.DeterministicReviewProvider{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(`{"project_url":"https://gitlab.com/acme/widget","mr_iid":7}`))

	NewRouterWithDeps(RouterDeps{Reviews: service, Broker: broker}).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", rec.Code, rec.Body.String())
	}
}

func TestReviewEventsSSEStreamsEventNameAndJSONPayload(t *testing.T) {
	broker := events.NewBroker()
	service, _ := testReviewService(t, provider.DeterministicReviewProvider{})
	router := NewRouterWithDeps(RouterDeps{Reviews: service, Broker: broker})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/reviews/review_123/events", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { router.ServeHTTP(rec, req); close(done) }()
	time.Sleep(10 * time.Millisecond)
	broker.Publish("review_123", "agent.started", map[string]any{"review_id": "review_123", "dimension": "risk"})
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	scanner := bufio.NewScanner(strings.NewReader(rec.Body.String()))
	foundEvent, foundData := false, false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "event: agent.started" {
			foundEvent = true
		}
		if strings.HasPrefix(line, "data: ") {
			var payload map[string]any
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				t.Fatalf("data is not JSON: %v", err)
			}
			foundData = payload["review_id"] == "review_123" && payload["dimension"] == "risk"
		}
	}
	if !foundEvent || !foundData {
		t.Fatalf("SSE output missing event/data: %q", rec.Body.String())
	}
}

func TestReviewEventsSSEStreamsReviewAndAgentEventPayloads(t *testing.T) {
	broker := events.NewBroker()
	service, _ := testReviewService(t, provider.DeterministicReviewProvider{})
	router := NewRouterWithDeps(RouterDeps{Reviews: service, Broker: broker})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/reviews/review_coverage/events", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { router.ServeHTTP(rec, req); close(done) }()
	time.Sleep(10 * time.Millisecond)

	eventsToPublish := []struct {
		name    string
		payload map[string]any
	}{
		{name: "review.started", payload: map[string]any{"review_id": "review_coverage", "status": reviews.StatusRunning}},
		{name: "agent.started", payload: map[string]any{"review_id": "review_coverage", "dimension": "risk"}},
		{name: "agent.completed", payload: map[string]any{"review_id": "review_coverage", "dimension": "risk", "attempts": 1}},
		{name: "agent.error", payload: map[string]any{"review_id": "review_coverage", "dimension": "reliability", "error": map[string]any{"code": "PROVIDER_ERROR", "message": "provider down"}}},
		{name: "review.generated", payload: map[string]any{"review_id": "review_coverage", "status": reviews.StatusAwaitingApproval, "comments": 4, "verdict": "pass"}},
		{name: "review.error", payload: map[string]any{"review_id": "review_coverage", "status": reviews.StatusError, "errors": []any{map[string]any{"dimension": "platform", "code": "PLATFORM_ERROR", "message": "gitlab unavailable"}}}},
	}
	for _, event := range eventsToPublish {
		broker.Publish("review_coverage", event.name, event.payload)
	}
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	got := parseSSEEvents(t, rec.Body.String())
	for _, want := range eventsToPublish {
		payload, ok := got[want.name]
		if !ok {
			t.Fatalf("missing event %s in SSE output: %q", want.name, rec.Body.String())
		}
		if payload["review_id"] != "review_coverage" {
			t.Fatalf("event %s review_id = %v, want review_coverage", want.name, payload["review_id"])
		}
		switch want.name {
		case "review.started", "review.generated", "review.error":
			if _, ok := payload["status"].(string); !ok {
				t.Fatalf("event %s missing string status: %+v", want.name, payload)
			}
		case "agent.started", "agent.completed", "agent.error":
			if _, ok := payload["dimension"].(string); !ok {
				t.Fatalf("event %s missing string dimension: %+v", want.name, payload)
			}
		}
	}
}

func parseSSEEvents(t *testing.T, body string) map[string]map[string]any {
	t.Helper()
	parsed := map[string]map[string]any{}
	scanner := bufio.NewScanner(strings.NewReader(body))
	current := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			current = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") && current != "" {
			var payload map[string]any
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				t.Fatalf("event %s data is not JSON: %v", current, err)
			}
			parsed[current] = payload
			current = ""
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan SSE output: %v", err)
	}
	return parsed
}

func testReviewService(t *testing.T, p provider.ModelProvider) (*reviews.Service, *events.Broker) {
	return testReviewServiceWithPlatform(t, fakeAPIPlatform{}, p)
}

func testReviewServiceWithPlatform(t *testing.T, platformClient platform.PlatformClient, p provider.ModelProvider) (*reviews.Service, *events.Broker) {
	t.Helper()
	broker := events.NewBroker()
	return &reviews.Service{Repo: reviews.NewRepository(reviewsTestDB(t)), Platform: platformClient, Provider: p, Broker: broker}, broker
}

func reviewsTestDB(t *testing.T) *sql.DB {
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

type fakeAPIPlatform struct{}

func (f fakeAPIPlatform) InferProject(rawURL string) (platform.ProjectIdentity, error) {
	return platform.ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: "acme/widget", WebURL: rawURL}, nil
}

type failingAPIPlatform struct{}

func (f failingAPIPlatform) InferProject(rawURL string) (platform.ProjectIdentity, error) {
	return platform.ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: "acme/widget", WebURL: rawURL}, nil
}

func (f failingAPIPlatform) FetchMergeRequestContext(ctx context.Context, projectPath string, mrIID int) (platform.MergeRequestContext, error) {
	return platform.MergeRequestContext{}, errors.New("gitlab unavailable")
}

func (f fakeAPIPlatform) FetchMergeRequestContext(ctx context.Context, projectPath string, mrIID int) (platform.MergeRequestContext, error) {
	return platform.MergeRequestContext{
		Project:  platform.ProjectIdentity{Platform: "gitlab", Host: "gitlab.com", Path: projectPath, WebURL: "https://gitlab.com/" + projectPath},
		MR:       platform.MergeRequestMetadata{IID: mrIID, Title: "Improve widgets", WebURL: "https://gitlab.com/acme/widget/-/merge_requests/7"},
		BaseSHA:  "base-sha",
		StartSHA: "start-sha",
		HeadSHA:  "head-sha",
		Files:    []platform.ChangedFile{{NewPath: "README.md", Positions: []platform.DiffPosition{{NewLine: 1}}}},
	}, nil
}

type failingAPIProvider struct{}

func (f failingAPIProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	return provider.CompletionResponse{}, errors.New("provider down")
}
func (f failingAPIProvider) Name() string                          { return "failing" }
func (f failingAPIProvider) SupportedModels() []provider.ModelInfo { return nil }
