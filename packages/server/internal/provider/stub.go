package provider

import (
	"context"
	"fmt"
)

// NotImplementedProvider reserves real vendor names without linking SDKs yet.
type NotImplementedProvider struct {
	name  string
	model string
}

func NewNotImplementedProvider(name string, model string) *NotImplementedProvider {
	return &NotImplementedProvider{name: normalizeName(name), model: model}
}

func (p *NotImplementedProvider) Complete(context.Context, CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{}, fmt.Errorf("provider %q is not implemented; real SDK integration belongs to a later phase", p.name)
}

func (p *NotImplementedProvider) Name() string { return p.name }

func (p *NotImplementedProvider) SupportedModels() []ModelInfo {
	if p.model == "" {
		return nil
	}
	return []ModelInfo{{ID: p.model, DisplayName: p.model}}
}
