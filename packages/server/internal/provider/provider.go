package provider

import "context"

// CompletionRequest contains the prompt data sent to a model provider.
type CompletionRequest struct {
	System    string
	User      string
	MaxTokens int
	Provider  string
	Model     string
}

// CompletionResponse contains model output and token metadata.
type CompletionResponse struct {
	Content      string
	ModelUsed    string
	InputTokens  int
	OutputTokens int
}

// TokenTotal returns total token usage reported by the provider.
func (r CompletionResponse) TokenTotal() int {
	return r.InputTokens + r.OutputTokens
}

// ModelInfo describes a model exposed by a provider.
type ModelInfo struct {
	ID              string
	DisplayName     string
	ContextWindow   int
	CostPer1kInput  float64
	CostPer1kOutput float64
}

// ModelProvider is the boundary between the review harness and AI vendors.
type ModelProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Name() string
	SupportedModels() []ModelInfo
}
