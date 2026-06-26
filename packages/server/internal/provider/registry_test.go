package provider

import (
	"context"
	"errors"
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
	r := DefaultRegistry()
	got, err := r.Resolve(ProviderConfig{Name: "claude", ModelName: "claude-sonnet"})
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if got.Name() != "claude" {
		t.Fatalf("provider name = %q, want claude", got.Name())
	}
	if _, err := got.Complete(context.Background(), CompletionRequest{}); err == nil {
		t.Fatal("stub Complete() error = nil, want not implemented error")
	}
}
