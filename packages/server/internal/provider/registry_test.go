package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegistryResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		register  bool
		lookup    string
		wantName  string
		wantError error
	}{
		{name: "registered exact", register: true, lookup: "fake", wantName: "fake"},
		{name: "registered normalized", register: true, lookup: "  FAKE  ", wantName: "fake"},
		{name: "missing provider", lookup: "missing", wantError: ErrProviderNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			if tt.register {
				if err := r.Register("fake", func(ProviderConfig) (ModelProvider, error) {
					return NewFake(), nil
				}); err != nil {
					t.Fatalf("register provider: %v", err)
				}
			}

			got, err := r.Resolve(ProviderConfig{Name: tt.lookup})
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("Resolve() error = %v, want %v", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() unexpected error: %v", err)
			}
			if got.Name() != tt.wantName {
				t.Fatalf("provider name = %q, want %q", got.Name(), tt.wantName)
			}
		})
	}
}

func TestDefaultRegistryUsesStubs(t *testing.T) {
	t.Parallel()
	wantNames := []string{"claude", "gemini", "groq", "mistral", "ollama", "openai"}
	r := DefaultRegistry()
	gotNames := r.Names()
	if len(gotNames) != len(wantNames) {
		t.Fatalf("registered names = %v, want %v", gotNames, wantNames)
	}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Fatalf("registered names = %v, want %v", gotNames, wantNames)
		}
	}

	for _, name := range wantNames {
		t.Run(name, func(t *testing.T) {
			got, err := r.Resolve(ProviderConfig{Name: name, ModelName: name + "-model"})
			if err != nil {
				t.Fatalf("Resolve() unexpected error: %v", err)
			}
			if got.Name() != name {
				t.Fatalf("provider name = %q, want %q", got.Name(), name)
			}
			models := got.SupportedModels()
			if len(models) != 1 || models[0].ID != name+"-model" {
				t.Fatalf("SupportedModels() = %+v, want configured model", models)
			}
			_, err = got.Complete(context.Background(), CompletionRequest{})
			if err == nil || !strings.Contains(err.Error(), "not implemented") {
				t.Fatalf("stub Complete() error = %v, want not implemented error", err)
			}
		})
	}
}

func TestRegistryDuplicateRegistrationReplacesFactory(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register("fake", func(ProviderConfig) (ModelProvider, error) {
		return &FakeProvider{NameValue: "first"}, nil
	}); err != nil {
		t.Fatalf("register first provider: %v", err)
	}
	if err := r.Register(" FAKE ", func(ProviderConfig) (ModelProvider, error) {
		return &FakeProvider{NameValue: "second"}, nil
	}); err != nil {
		t.Fatalf("register replacement provider: %v", err)
	}

	got, err := r.Resolve(ProviderConfig{Name: "fake"})
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if got.Name() != "second" {
		t.Fatalf("provider name = %q, want replacement factory", got.Name())
	}
}

func TestFakeProviderRecordsRequestsAndConsumesQueues(t *testing.T) {
	t.Parallel()
	fake := NewFake(
		CompletionResponse{Content: "first"},
		CompletionResponse{Content: "second"},
	)
	fake.QueueError(errors.New("queued failure"))

	if _, err := fake.Complete(context.Background(), CompletionRequest{System: "s1", User: "u1"}); err == nil {
		t.Fatal("first Complete() error = nil, want queued error")
	}
	resp, err := fake.Complete(context.Background(), CompletionRequest{System: "s2", User: "u2"})
	if err != nil {
		t.Fatalf("second Complete() error = %v", err)
	}
	if resp.Content != "first" {
		t.Fatalf("second response content = %q, want first queued response", resp.Content)
	}
	resp, err = fake.Complete(context.Background(), CompletionRequest{System: "s3", User: "u3"})
	if err != nil {
		t.Fatalf("third Complete() error = %v", err)
	}
	if resp.Content != "second" {
		t.Fatalf("third response content = %q, want second queued response", resp.Content)
	}

	requests := fake.Requests()
	if len(requests) != 3 {
		t.Fatalf("recorded requests = %d, want 3", len(requests))
	}
	for i, want := range []string{"s1", "s2", "s3"} {
		if requests[i].System != want {
			t.Fatalf("request %d system = %q, want %q", i, requests[i].System, want)
		}
	}
}
