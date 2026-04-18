// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"slices"
	"time"

	core "dappco.re/go/core"
)

// provider := agentic.NewProviderManager(nil).Provider("claude")
//
//	core.Println(provider.Name()) // "claude"
type AgenticProviderInterface interface {
	Generate(context.Context, string, map[string]any) (string, error)
	Stream(context.Context, string, map[string]any, func(string)) error
	Name() string
	DefaultModel() string
	IsAvailable() bool
}

// manager := agentic.NewProviderManager(nil)
// core.Println(manager.Names()) // ["claude", "gemini", "openai"]
type ProviderManager struct {
	providers map[string]AgenticProviderInterface
}

var providerRetryBaseDelay = 100 * time.Millisecond
var providerSleep = time.Sleep

const providerRetryAttempts = 3

// manager := s.providerManager()
// core.Println(manager.Names()) // ["claude", "gemini", "openai"]
func (s *PrepSubsystem) providerManager() *ProviderManager {
	if s == nil {
		return NewProviderManager(nil)
	}
	if s.providers != nil {
		return s.providers
	}

	s.providers = NewProviderManager(func(ctx context.Context, prompt string, options map[string]any) (string, error) {
		config := anyMapValue(options["config"])
		if model := contentMapStringValue(options, "model"); model != "" {
			if config == nil {
				config = map[string]any{}
			}
			config["model"] = model
		}
		input := ContentGenerateInput{
			Prompt:   prompt,
			Provider: contentMapStringValue(options, "provider"),
			Config:   config,
		}
		if template := contentMapStringValue(options, "template"); template != "" {
			input.Template = template
		}
		if briefID := contentMapStringValue(options, "brief_id", "briefId"); briefID != "" {
			input.BriefID = briefID
		}
		result, err := s.contentGenerateResult(ctx, input)
		if err != nil {
			return "", err
		}
		return result.Content, nil
	})

	return s.providers
}

//	manager := agentic.NewProviderManager(func(ctx context.Context, prompt string, options map[string]any) (string, error) {
//	    return "Draft ready", nil
//	})
//
// core.Println(manager.Names()) // ["claude", "gemini", "openai"]
func NewProviderManager(generate ProviderGenerateFunc) *ProviderManager {
	manager := &ProviderManager{
		providers: make(map[string]AgenticProviderInterface),
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
//	_ = provider.Stream(ctx, "Draft a release note", nil, func(token string) { core.Print(nil, token) })
type ProviderStreamFunc func(context.Context, string, map[string]any, func(string)) error

type contentProvider struct {
	name         string
	defaultModel string
	available    bool
	generate     ProviderGenerateFunc
	stream       ProviderStreamFunc
}

func newContentProvider(name, defaultModel string, available bool, generate ProviderGenerateFunc) *contentProvider {
	provider := &contentProvider{
		name:         name,
		defaultModel: defaultModel,
		available:    available,
		generate:     generate,
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
	return provider
}

func (p *contentProvider) Generate(ctx context.Context, prompt string, options map[string]any) (string, error) {
	if p.generate == nil {
		return "", core.E("provider.generate", core.Concat("provider not configured: ", p.name), nil)
	}

	var lastErr error
	delay := providerRetryBaseDelay
	for attempt := 1; attempt <= providerRetryAttempts; attempt++ {
		optionsCopy := map[string]any{}
		for key, value := range options {
			optionsCopy[key] = value
		}
		if optionsCopy["provider"] == nil {
			optionsCopy["provider"] = p.name
		}
		if optionsCopy["model"] == nil && p.defaultModel != "" {
			optionsCopy["model"] = p.defaultModel
		}

		content, err := p.generate(ctx, prompt, optionsCopy)
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

func (p *contentProvider) Stream(ctx context.Context, prompt string, options map[string]any, onToken func(string)) error {
	if p.stream == nil {
		return core.E("provider.stream", core.Concat("provider not configured: ", p.name), nil)
	}
	return p.stream(ctx, prompt, options, onToken)
}

func (p *contentProvider) Name() string {
	return p.name
}

func (p *contentProvider) DefaultModel() string {
	return p.defaultModel
}

func (p *contentProvider) IsAvailable() bool {
	return p.available
}

// Register adds or replaces a provider in the registry.
//
//	manager.Register(newContentProvider("claude", "claude-3.7-sonnet", true, generate))
func (m *ProviderManager) Register(provider AgenticProviderInterface) {
	if m == nil || provider == nil {
		return
	}
	if m.providers == nil {
		m.providers = make(map[string]AgenticProviderInterface)
	}
	m.providers[core.Lower(core.Trim(provider.Name()))] = provider
}

// Provider returns a registered provider by name.
//
//	provider, ok := manager.Provider("openai")
func (m *ProviderManager) Provider(name string) (AgenticProviderInterface, bool) {
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
func (m *ProviderManager) DefaultProvider() AgenticProviderInterface {
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
