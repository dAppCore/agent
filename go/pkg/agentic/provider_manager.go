// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"slices"
	"time"

	core "dappco.re/go"
)

// provider := agentic.NewProviderManager(nil).Provider("claude")
//
//	core.Println(provider.Name()) // "claude"
type AgenticProviderInterface struct {
	Generate     ProviderGenerateFunc
	Stream       ProviderStreamFunc
	Name         func() string
	DefaultModel func() string
	IsAvailable  func() bool
	available    bool
	stream       ProviderStreamFunc
}

// manager := agentic.NewProviderManager(nil)
// core.Println(manager.Names()) // ["claude", "gemini", "openai"]
type ProviderManager struct {
	providers map[string]*AgenticProviderInterface
}

var providerRetryBaseDelay = 100 * time.Millisecond
var providerSleep = time.Sleep

const providerRetryAttempts = 3

//	manager := agentic.NewProviderManager(func(ctx context.Context, prompt string, options map[string]any) (string, error) {
//	    return "Draft ready", nil
//	})
//
// core.Println(manager.Names()) // ["claude", "gemini", "openai"]
func NewProviderManager(generate ProviderGenerateFunc) *ProviderManager {
	manager := &ProviderManager{
		providers: make(map[string]*AgenticProviderInterface),
	}

	manager.Register(newContentProvider("claude", "claude-3.7-sonnet", true, generate))
	manager.Register(newContentProvider("gemini", "gemini-2.5-pro", true, generate))
	manager.Register(newContentProvider("openai", "gpt-5.4", true, generate))

	return manager
}

// provider, _ := manager.Provider("claude")
// text, _ := provider.Generate(ctx, "Draft a release note", map[string]any{"temperature": 0.2})
type ProviderGenerateFunc func(context.Context, string, map[string]any) (string, error)

// Stream sends provider output to the callback as it arrives.
//
//	provider, _ := manager.Provider("claude")
//	result := provider.Stream(ctx, "Draft a release note", nil, func(token string) { core.Print(nil, token) })
type ProviderStreamFunc func(context.Context, string, map[string]any, func(string)) error

func newContentProvider(name, defaultModel string, available bool, generate ProviderGenerateFunc) *AgenticProviderInterface {
	provider := &AgenticProviderInterface{}
	provider.available = available
	provider.Name = func() string {
		return name
	}
	provider.DefaultModel = func() string {
		return defaultModel
	}
	provider.IsAvailable = func() bool {
		return provider.available
	}
	provider.Generate = func(ctx context.Context, prompt string, options map[string]any) (string, error) {
		if generate == nil {
			return "", core.E("provider.generate", core.Concat("provider not configured: ", name), nil)
		}

		var lastErr error
		delay := providerRetryBaseDelay
		for attempt := 1; attempt <= providerRetryAttempts; attempt++ {
			optionsCopy := map[string]any{}
			for key, value := range options {
				optionsCopy[key] = value
			}
			if optionsCopy["provider"] == nil {
				optionsCopy["provider"] = name
			}
			if optionsCopy["model"] == nil && defaultModel != "" {
				optionsCopy["model"] = defaultModel
			}

			content, err := generate(ctx, prompt, optionsCopy)
			if err == nil {
				return content, nil
			}
			lastErr = err
			if attempt == providerRetryAttempts {
				break
			}
			if ctx != nil {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				default:
				}
			}
			if delay > 0 {
				providerSleep(delay)
				delay *= 2
				continue
			}
			delay *= 2
		}

		return "", lastErr
	}
	provider.stream = func(ctx context.Context, prompt string, options map[string]any, onToken func(string)) error {
		content, err := provider.Generate(ctx, prompt, options)
		if err != nil {
			return err
		}
		if onToken != nil {
			onToken(content)
		}
		return nil
	}
	provider.Stream = func(ctx context.Context, prompt string, options map[string]any, onToken func(string)) error {
		if provider.stream == nil {
			return core.E("provider.stream", core.Concat("provider not configured: ", name), nil)
		}
		return provider.stream(ctx, prompt, options, onToken)
	}
	return provider
}

// Register adds or replaces a provider in the registry.
//
//	manager.Register(newContentProvider("claude", "claude-3.7-sonnet", true, generate))
func (m *ProviderManager) Register(provider *AgenticProviderInterface) {
	if m == nil || provider == nil {
		return
	}
	if m.providers == nil {
		m.providers = make(map[string]*AgenticProviderInterface)
	}
	m.providers[core.Lower(core.Trim(provider.Name()))] = provider
}

// Provider returns a registered provider by name.
//
//	provider, ok := manager.Provider("openai")
func (m *ProviderManager) Provider(name string) (*AgenticProviderInterface, bool) {
	if m == nil {
		return nil, false
	}
	provider, ok := m.providers[core.Lower(core.Trim(name))]
	return provider, ok
}

// Names returns the registered provider names in deterministic order.
//
//	core.Println(manager.Names()) // ["claude", "gemini", "openai"]
func (m *ProviderManager) Names() []string {
	if m == nil || len(m.providers) == 0 {
		return nil
	}

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// DefaultProvider returns the first registered provider that is available.
//
//	provider := manager.DefaultProvider()
func (m *ProviderManager) DefaultProvider() *AgenticProviderInterface {
	if m == nil {
		return nil
	}

	for _, name := range m.Names() {
		if provider, ok := m.Provider(name); ok && provider.IsAvailable() {
			return provider
		}
	}

	return nil
}
