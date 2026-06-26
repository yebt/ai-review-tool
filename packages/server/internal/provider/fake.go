package provider

import (
	"context"
	"errors"
	"sync"
	"time"
)

// FakeProvider is a deterministic provider for unit tests and local harness checks.
type FakeProvider struct {
	NameValue string
	Models    []ModelInfo
	Delay     time.Duration

	mu        sync.Mutex
	responses []CompletionResponse
	errors    []error
	requests  []CompletionRequest
}

func NewFake(responses ...CompletionResponse) *FakeProvider {
	return &FakeProvider{NameValue: "fake", responses: responses}
}

func (p *FakeProvider) QueueResponse(resp CompletionResponse) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = append(p.responses, resp)
}

func (p *FakeProvider) QueueError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		err = errors.New("fake provider error")
	}
	p.errors = append(p.errors, err)
}

func (p *FakeProvider) Requests() []CompletionRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]CompletionRequest, len(p.requests))
	copy(out, p.requests)
	return out
}

func (p *FakeProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	if p.Delay > 0 {
		select {
		case <-time.After(p.Delay):
		case <-ctx.Done():
			return CompletionResponse{}, ctx.Err()
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, req)
	if len(p.errors) > 0 {
		err := p.errors[0]
		p.errors = p.errors[1:]
		return CompletionResponse{}, err
	}
	if len(p.responses) == 0 {
		return CompletionResponse{}, errors.New("fake provider has no queued response")
	}
	resp := p.responses[0]
	p.responses = p.responses[1:]
	return resp, nil
}

func (p *FakeProvider) Name() string {
	if p.NameValue == "" {
		return "fake"
	}
	return p.NameValue
}

func (p *FakeProvider) SupportedModels() []ModelInfo { return p.Models }
