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

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk"}, fake, AgentPrompt{})
	if result.Error != nil {
		t.Fatalf("Run() error = %+v", result.Error)
	}
	if result.Attempts != 1 || result.Tokens.Total != 30 || len(result.Output) == 0 {
		t.Fatalf("unexpected result: %+v", result)
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

func TestRunInvalidJSON(t *testing.T) {
	t.Parallel()
	fake := provider.NewFake(provider.CompletionResponse{Content: `not-json`})

	result := Run(context.Background(), Config{Dimension: "risk", Timeout: time.Second, OutputSchema: "risk"}, fake, AgentPrompt{})
	if result.Error == nil || result.Error.Code != ErrorInvalidJSON {
		t.Fatalf("Run() error = %+v, want invalid JSON", result.Error)
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

func validOutput(dimension string) string {
	return `{"dimension":"` + dimension + `","score":90,"findings":[{"severity":"WARNING","file":"main.go","line_start":1,"line_end":1,"evidence":"func main() {}","why":"This is specific enough for the output contract.","suggestion_snippet":"func main() {}","inline_comment":true}],"summary":"One finding.","verdict":"needs_changes"}`
}
