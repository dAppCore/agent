// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderManager_NewProviderManager_Good_RegistersBuiltIns(t *testing.T) {
	manager := NewProviderManager(func(context.Context, string, map[string]any) (string, error) {
		return "Draft ready", nil
	})

	require.NotNil(t, manager)
	assert.Equal(t, []string{"claude", "gemini", "openai"}, manager.Names())

	provider, ok := manager.Provider("claude")
	require.True(t, ok)
	assert.Equal(t, "claude", provider.Name())
	assert.Equal(t, "claude-3.7-sonnet", provider.DefaultModel())

	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	require.NoError(t, err)
	assert.Equal(t, "Draft ready", text)
}

func TestProviderManager_Provider_Bad_UnknownNameReturnsFalse(t *testing.T) {
	manager := NewProviderManager(nil)

	provider, ok := manager.Provider("unknown")
	assert.False(t, ok)
	assert.Nil(t, provider)
}

func TestProviderManager_ContentProvider_Ugly_NoGeneratorReturnsError(t *testing.T) {
	provider := newContentProvider("claude", "claude-3.7-sonnet", true, nil)

	_, err := provider.Generate(context.Background(), "Draft a release note", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider not configured")
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

		assert.Equal(t, "claude", options["provider"])
		assert.Equal(t, "claude-3.7-sonnet", options["model"])
		return "Draft ready", nil
	})

	text, err := provider.Generate(context.Background(), "Write a release note", nil)
	require.NoError(t, err)
	assert.Equal(t, "Draft ready", text)
	assert.Equal(t, 3, attempts)
	assert.Equal(t, []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}, delays)
}
