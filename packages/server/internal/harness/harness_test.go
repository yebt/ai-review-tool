package harness

import (
	"context"
	"errors"
	"testing"
	"time"

	"co-review/server/internal/provider"
)

func TestRunSuccess(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(provider.CompletionResponse{Content: validOutput("risk"), ModelUsed: "fake", InputTokens: 10, OutputTokens: 20})

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk", MaxTokens: 512}, fake, AgentPrompt{System: "system prompt", User: "user prompt"})
	if result.Error != nil {
		t.Fatalf("Run() error = %+v", result.Error)
	}
	if result.Attempts != 1 || result.Tokens.Total != 30 || len(result.Output) == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	requests := fake.Requests()
	if len(requests) != 1 {
		t.Fatalf("provider requests = %d, want 1", len(requests))
	}
	if requests[0].System != "system prompt" || requests[0].User != "user prompt" || requests[0].MaxTokens != 512 {
		t.Fatalf("provider request = %+v, want prompt and max tokens propagated", requests[0])
	}
}

func TestRunTimeout(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(provider.CompletionResponse{Content: validOutput("risk")})
	fake.Delay = 30 * time.Millisecond

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Millisecond, OutputSchema: "risk"}, fake, AgentPrompt{})
	if result.Error == nil || result.Error.Code != ErrorTimeout {
		t.Fatalf("Run() error = %+v, want timeout", result.Error)
	}
}

func TestRunProviderErrorRetry(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(provider.CompletionResponse{Content: validOutput("risk")})
	fake.QueueError(errors.New("temporary provider failure"))

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk", MaxRetries: 1}, fake, AgentPrompt{})
	if result.Error != nil {
		t.Fatalf("Run() error = %+v", result.Error)
	}
	if result.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", result.Attempts)
	}
}

func TestRunAttemptsContractExhaustsInitialAttemptPlusRetries(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake()
	fake.QueueError(errors.New("first failure"))
	fake.QueueError(errors.New("second failure"))
	fake.QueueError(errors.New("third failure"))

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk", MaxRetries: 2}, fake, AgentPrompt{})
	if result.Error == nil || result.Error.Code != ErrorProvider {
		t.Fatalf("Run() error = %+v, want provider error", result.Error)
	}
	if result.Attempts != 3 {
		t.Fatalf("attempts = %d, want initial attempt plus 2 retries", result.Attempts)
	}
	if requests := fake.Requests(); len(requests) != 3 {
		t.Fatalf("provider requests = %d, want 3", len(requests))
	}
}

func TestRunInvalidJSON(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(provider.CompletionResponse{Content: `not-json`})

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk"}, fake, AgentPrompt{})
	if result.Error == nil || result.Error.Code != ErrorInvalidJSON {
		t.Fatalf("Run() error = %+v, want invalid JSON", result.Error)
	}
}

func TestRunInvalidJSONRetriesAndCanRecover(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(
		provider.CompletionResponse{Content: `not-json`},
		provider.CompletionResponse{Content: validOutput("risk"), InputTokens: 2, OutputTokens: 3},
	)

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk", MaxRetries: 1}, fake, AgentPrompt{})
	if result.Error != nil {
		t.Fatalf("Run() error = %+v", result.Error)
	}
	if result.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", result.Attempts)
	}
	if result.Tokens.Total != 5 {
		t.Fatalf("token total = %d, want successful response total", result.Tokens.Total)
	}
}

func TestRunInvalidSchema(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(provider.CompletionResponse{Content: `{"dimension":"risk","findings":[],"summary":"ok","verdict":"maybe"}`})

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk"}, fake, AgentPrompt{})
	if result.Error == nil || result.Error.Code != ErrorInvalidSchema {
		t.Fatalf("Run() error = %+v, want invalid schema", result.Error)
	}
}

func TestRunInvalidSchemaRetriesAndCanRecover(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(
		provider.CompletionResponse{Content: `{"dimension":"risk","findings":[],"summary":"ok","verdict":"maybe"}`},
		provider.CompletionResponse{Content: validOutput("risk")},
	)

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk", MaxRetries: 1}, fake, AgentPrompt{})
	if result.Error != nil {
		t.Fatalf("Run() error = %+v", result.Error)
	}
	if result.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", result.Attempts)
	}
}

func TestRunContextCancellationStopsRetries(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake()
	fake.QueueError(errors.New("temporary provider failure"))
	fake.QueueError(errors.New("should not be reached"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := Run(ctx, Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk", MaxRetries: 3, RetryBackoff: time.Hour}, fake, AgentPrompt{})
	if result.Error == nil || result.Error.Code != ErrorProvider {
		t.Fatalf("Run() error = %+v, want provider error", result.Error)
	}
	if result.Attempts != 1 {
		t.Fatalf("attempts = %d, want cancellation to stop retries after first failure", result.Attempts)
	}
	if requests := fake.Requests(); len(requests) != 1 {
		t.Fatalf("provider requests = %d, want 1", len(requests))
	}
}

func validOutput(dimension string) string {
	return `{"dimension":"` + dimension + `","score":90,"findings":[{"severity":"WARNING","file":"main.go","line_start":1,"line_end":1,"evidence":"func main() {}","why":"This is specific enough for the output contract.","suggestion_snippet":"func main() {}","inline_comment":true}],"summary":"One finding.","verdict":"needs_changes"}`
}
