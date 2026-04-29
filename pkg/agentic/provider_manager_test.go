// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestProviderManager_NewProviderManager_Good_RegistersBuiltIns(t *testing.T) {
	manager := NewProviderManager(func(context.Context, string, map[string]any) (string, error) {
		return "Draft ready", nil
	})

	core.AssertNotNil(t, manager)
	core.AssertEqual(t, []string{"claude", "gemini", "openai"}, manager.Names())

	provider, ok := manager.Provider("claude")
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "claude", provider.Name())
	core.AssertEqual(t, "claude-3.7-sonnet", provider.DefaultModel())

	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Draft ready", text)
}

func TestProviderManager_Provider_Bad_UnknownNameReturnsFalse(t *testing.T) {
	manager := NewProviderManager(nil)

	provider, ok := manager.Provider("unknown")
	core.AssertFalse(t, ok)
	core.AssertNil(t, provider)
}

func TestProviderManager_ContentProvider_Ugly_NoGeneratorReturnsError(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)

	_, err := provider.Generate(context.Background(), "Draft a release note", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "provider not configured")
}

func TestProviderManager_ContentProvider_Good_RetriesWithExponentialBackoff(t *testing.T) {
	originalSleep := providerSleep
	originalDelay := providerRetryBaseDelay
	defer func() {
		providerSleep = originalSleep
		providerRetryBaseDelay = originalDelay
	}()

	var delays []time.Duration
	providerSleep = func(delay time.Duration) {
		delays = append(delays, delay)
	}
	providerRetryBaseDelay = 50 * time.Millisecond

	attempts := 0
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, func(_ context.Context, _ string, options map[string]any) (string, error) {
		attempts++
		if attempts < 3 {
			return "", core.E("test.generate", "transient failure", nil)
		}

		core.AssertEqual(t, "claude", options["provider"])
		core.AssertEqual(t, "claude-3.7-sonnet", options["model"])
		return "Draft ready", nil
	})

	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Draft ready", text)
	core.AssertEqual(t, 3, attempts)
	core.AssertEqual(t, []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}, delays)
}

func TestProviderManager_NewProviderManager_Good(t *testing.T) {
	manager := NewProviderManager(func(context.Context, string, map[string]any) (string, error) {
		return "Draft ready", nil
	})

	core.AssertNotNil(t, manager)
	core.AssertEqual(t, []string{"claude", "gemini", "openai"}, manager.Names())

	provider, ok := manager.Provider("claude")
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "claude", provider.Name())
	core.AssertEqual(t, "claude-3.7-sonnet", provider.DefaultModel())

	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Draft ready", text)
}

func TestProviderManager_NewProviderManager_Bad(t *testing.T) {
	manager := NewProviderManager(nil)
	core.AssertNotNil(t, manager)
	core.AssertContains(t, manager.Names(), "openai")
}

func TestProviderManager_NewProviderManager_Ugly(t *testing.T) {
	first := NewProviderManager(nil)
	second := NewProviderManager(nil)
	core.AssertTrue(t, first != second)
	core.AssertEqual(t, first.Names(), second.Names())
}

func TestProviderManager_Provider_Generate_Good(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, func(_ context.Context, _ string, options map[string]any) (string, error) {
		core.AssertEqual(t, "claude", options["provider"])
		return "Draft ready", nil
	})
	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Draft ready", text)
}

func TestProviderManager_Provider_Generate_Bad(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)
	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	core.AssertEqual(t, "", text)
	core.AssertError(t, err)
}

func TestProviderManager_Provider_Generate_Ugly(t *testing.T) {
	attempts := 0
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, func(_ context.Context, _ string, _ map[string]any) (string, error) {
		attempts++
		if attempts == 1 {
			return "", core.E("test.generate", "transient failure", nil)
		}
		return "Draft ready", nil
	})
	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Draft ready", text)
}

func TestProviderManager_Provider_Stream_Good(t *testing.T) {
	var tokens []string
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, func(_ context.Context, _ string, _ map[string]any) (string, error) {
		return "Draft ready", nil
	})
	err := provider.Stream(context.Background(), "Write a release note", nil, func(token string) {
		tokens = append(tokens, token)
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, []string{"Draft ready"}, tokens)
}

func TestProviderManager_Provider_Stream_Bad(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)
	provider.stream = nil
	err := provider.Stream(context.Background(), "Write a release note", nil, nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "provider not configured")
}

func TestProviderManager_Provider_Stream_Ugly(t *testing.T) {
	calls := 0
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, func(_ context.Context, _ string, _ map[string]any) (string, error) {
		calls++
		return "Draft ready", nil
	})
	err := provider.Stream(context.Background(), "Write a release note", nil, nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, calls)
}

func TestProviderManager_Provider_Name_Good(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)
	got := provider.Name()
	core.AssertEqual(t, "claude", got)
	core.AssertNotEmpty(t, got)
}

func TestProviderManager_Provider_Name_Bad(t *testing.T) {
	provider := newContentProvider("", "claude-3.7-sonnet", true, nil)
	got := provider.Name()
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestProviderManager_Provider_Name_Ugly(t *testing.T) {
	provider := newContentProvider("cl\u00e1ude", "claude-3.7-sonnet", true, nil)
	got := provider.Name()
	core.AssertEqual(t, "cl\u00e1ude", got)
	core.AssertContains(t, got, "\u00e1")
}

func TestProviderManager_Provider_DefaultModel_Good(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)
	got := provider.DefaultModel()
	core.AssertEqual(t, "claude-3.7-sonnet", got)
	core.AssertContains(t, got, "sonnet")
}

func TestProviderManager_Provider_DefaultModel_Bad(t *testing.T) {
	provider := newContentProvider("claude", "", true, nil)
	got := provider.DefaultModel()
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestProviderManager_Provider_DefaultModel_Ugly(t *testing.T) {
	provider := newContentProvider("claude", "m\u00f6del", true, nil)
	got := provider.DefaultModel()
	core.AssertEqual(t, "m\u00f6del", got)
	core.AssertContains(t, got, "\u00f6")
}

func TestProviderManager_Provider_IsAvailable_Good(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)
	core.AssertTrue(t, provider.IsAvailable())
	core.AssertEqual(t, true, provider.IsAvailable())
}

func TestProviderManager_Provider_IsAvailable_Bad(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", false, nil)
	core.AssertFalse(t, provider.IsAvailable())
	core.AssertEqual(t, false, provider.IsAvailable())
}

func TestProviderManager_Provider_IsAvailable_Ugly(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)
	provider.available = false
	core.AssertFalse(t, provider.IsAvailable())
	core.AssertEqual(t, false, provider.IsAvailable())
}

func TestProviderManager_ProviderManager_Register_Good(t *testing.T) {
	manager := &ProviderManager{}
	provider := newContentProvider("custom", "gpt-5.4", true, nil)
	manager.Register(provider)
	core.AssertContains(t, manager.Names(), "custom")
	core.AssertEqual(t, provider, manager.DefaultProvider())
}

func TestProviderManager_ProviderManager_Register_Bad(t *testing.T) {
	manager := &ProviderManager{}
	manager.Register(nil)
	core.AssertNil(t, manager.Names())
	core.AssertNil(t, manager.DefaultProvider())
}

func TestProviderManager_ProviderManager_Register_Ugly(t *testing.T) {
	manager := &ProviderManager{}
	first := newContentProvider("custom", "v1", true, nil)
	second := newContentProvider("custom", "v2", true, nil)
	manager.Register(first)
	manager.Register(second)
	provider, ok := manager.Provider("custom")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "v2", provider.DefaultModel())
}

func TestProviderManager_ProviderManager_Provider_Good(t *testing.T) {
	manager := NewProviderManager(nil)
	provider, ok := manager.Provider("CLAUDE")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "claude", provider.Name())
}

func TestProviderManager_ProviderManager_Provider_Bad(t *testing.T) {
	manager := NewProviderManager(nil)
	provider, ok := manager.Provider("unknown")
	core.AssertFalse(t, ok)
	core.AssertNil(t, provider)
}

func TestProviderManager_ProviderManager_Provider_Ugly(t *testing.T) {
	var manager *ProviderManager
	provider, ok := manager.Provider("claude")
	core.AssertFalse(t, ok)
	core.AssertNil(t, provider)
}

func TestProviderManager_ProviderManager_Names_Good(t *testing.T) {
	manager := NewProviderManager(nil)
	got := manager.Names()
	core.AssertEqual(t, []string{"claude", "gemini", "openai"}, got)
	core.AssertLen(t, got, 3)
}

func TestProviderManager_ProviderManager_Names_Bad(t *testing.T) {
	manager := &ProviderManager{}
	got := manager.Names()
	core.AssertNil(t, got)
	core.AssertLen(t, got, 0)
}

func TestProviderManager_ProviderManager_Names_Ugly(t *testing.T) {
	var manager *ProviderManager
	got := manager.Names()
	core.AssertNil(t, got)
	core.AssertLen(t, got, 0)
}

func TestProviderManager_ProviderManager_DefaultProvider_Good(t *testing.T) {
	manager := &ProviderManager{}
	manager.Register(newContentProvider("custom", "gpt-5.4", true, nil))
	provider := manager.DefaultProvider()
	core.AssertNotNil(t, provider)
	core.AssertEqual(t, "custom", provider.Name())
}

func TestProviderManager_ProviderManager_DefaultProvider_Bad(t *testing.T) {
	manager := &ProviderManager{}
	manager.Register(newContentProvider("custom", "gpt-5.4", false, nil))
	provider := manager.DefaultProvider()
	core.AssertNil(t, provider)
	core.AssertFalse(t, provider != nil)
}

func TestProviderManager_ProviderManager_DefaultProvider_Ugly(t *testing.T) {
	var manager *ProviderManager
	provider := manager.DefaultProvider()
	core.AssertNil(t, provider)
	core.AssertFalse(t, provider != nil)
}
