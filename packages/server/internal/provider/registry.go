package provider

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrProviderNotFound = errors.New("provider not found")

// ProviderConfig contains configuration needed to construct a provider.
type ProviderConfig struct {
	Name        string
	APIKey      string
	ModelName   string
	ExtraParams map[string]string
}

// Factory constructs a provider from runtime configuration.
type Factory func(ProviderConfig) (ModelProvider, error)

// Registry resolves provider names to provider factories.
type Registry struct {
	factories map[string]Factory
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// DefaultRegistry exposes known provider names without performing real API calls.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, name := range []string{"claude", "openai", "groq", "ollama", "gemini", "mistral"} {
		providerName := name
		r.Register(providerName, func(cfg ProviderConfig) (ModelProvider, error) {
			return NewNotImplementedProvider(providerName, cfg.ModelName), nil
		})
	}
	return r
}

// Register adds or replaces a provider factory by normalized name.
func (r *Registry) Register(name string, factory Factory) error {
	if r == nil {
		return errors.New("registry is nil")
	}
	name = normalizeName(name)
	if name == "" {
		return errors.New("provider name must not be empty")
	}
	if factory == nil {
		return errors.New("provider factory must not be nil")
	}
	r.factories[name] = factory
	return nil
}

// Resolve constructs a provider for the requested name.
func (r *Registry) Resolve(cfg ProviderConfig) (ModelProvider, error) {
	if r == nil {
		return nil, errors.New("registry is nil")
	}
	name := normalizeName(cfg.Name)
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, cfg.Name)
	}
	cfg.Name = name
	return factory(cfg)
}

// Names returns registered provider names in deterministic order.
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
