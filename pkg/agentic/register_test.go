// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Register ---

func TestRegister_ServiceRegistered_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_BRAIN_URL", "")

	c := core.New(core.WithService(Register))
	require.NotNil(t, c)

	// Service auto-registered under the last segment of the package path: "agentic"
	service := c.Service("agentic")
	require.True(t, service.OK)
	prep, ok := service.Value.(*PrepSubsystem)
	require.True(t, ok, "PrepSubsystem must be registered as \"agentic\"")
	assert.NotNil(t, prep)
}

func TestRegister_CoreWired_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	service := c.Service("agentic")
	require.True(t, service.OK)
	prep, ok := service.Value.(*PrepSubsystem)
	require.True(t, ok)
	// Register must wire ServiceRuntime — service needs it for Core access
	assert.NotNil(t, prep.ServiceRuntime, "Register must set ServiceRuntime")
	assert.Equal(t, c, prep.Core())
}

func TestRegister_AgentsConfig_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	// Register stores agents.concurrency into Core Config — verify it is present
	concurrency := core.ConfigGet[map[string]ConcurrencyLimit](c.Config(), "agents.concurrency")
	assert.NotNil(t, concurrency, "Register must store agents.concurrency in Core Config")
}
