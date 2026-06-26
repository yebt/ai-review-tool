package provider

import (
	"context"
	"fmt"
	"strings"
)

// DeterministicReviewProvider returns valid 4R JSON without calling external AI providers.
type DeterministicReviewProvider struct{}

func (p DeterministicReviewProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	select {
	case <-ctx.Done():
		return CompletionResponse{}, ctx.Err()
	default:
	}
	dimension := detectDimension(req.System + "\n" + req.User)
	content := fmt.Sprintf(`{"dimension":%q,"score":82,"findings":[{"severity":"SUGGESTION","file":"README.md","line_start":1,"line_end":1,"evidence":"Deterministic fake review evidence for %s.","why":"This deterministic finding keeps the Phase 4 review path testable without live providers.","suggestion_snippet":"Consider documenting this behavior.","inline_comment":true}],"summary":"Deterministic %s review completed.","verdict":"pass"}`, dimension, dimension, dimension)
	return CompletionResponse{Content: content, ModelUsed: "fake-deterministic", InputTokens: 64, OutputTokens: 96}, nil
}

func (p DeterministicReviewProvider) Name() string { return "fake-deterministic" }

func (p DeterministicReviewProvider) SupportedModels() []ModelInfo {
	return []ModelInfo{{ID: "fake-deterministic", DisplayName: "Fake deterministic", ContextWindow: 8192}}
}

func detectDimension(text string) string {
	for _, dimension := range []string{"risk", "readability", "reliability", "resilience"} {
		if strings.Contains(strings.ToLower(text), dimension) {
			return dimension
		}
	}
	return "risk"
}
