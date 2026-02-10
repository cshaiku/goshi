package cli

import (
	"fmt"

	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/llm/ollama"
	"github.com/cshaiku/goshi/internal/llm/openai"
)

// BackendFactory creates LLM backend instances
// Implements Dependency Inversion: high-level code depends on Backend interface,
// not concrete implementations. Placed in CLI package to avoid import cycles.
type BackendFactory struct {
	provider string
	model    string
}

// NewBackendFactory creates a factory for the specified provider
func NewBackendFactory(provider, model string) *BackendFactory {
	// Normalize provider name
	if provider == "" || provider == "auto" {
		provider = "ollama"
	}

	return &BackendFactory{
		provider: provider,
		model:    model,
	}
}

// Create instantiates the appropriate backend implementation
// Returns Backend interface, maintaining abstraction
func (f *BackendFactory) Create() (llm.Backend, error) {
	switch f.provider {
	case "ollama":
		return ollama.New(f.model), nil

	case "openai":
		return openai.New(f.model)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: ollama, openai)", f.provider)
	}
}

// SupportedProviders returns list of available providers
func SupportedProviders() []string {
	return []string{"ollama", "openai"}
}
