// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Register ---

func TestRegister_Good_ServiceRegistered(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_BRAIN_URL", "")

	c := core.New(core.WithService(Register))
	require.NotNil(t, c)

	// Service auto-registered under the last segment of the package path: "agentic"
	prep, ok := core.ServiceFor[*PrepSubsystem](c, "agentic")
	assert.True(t, ok, "PrepSubsystem must be registered as \"agentic\"")
	assert.NotNil(t, prep)
}

func TestRegister_Good_CoreWired(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	prep, ok := core.ServiceFor[*PrepSubsystem](c, "agentic")
	require.True(t, ok)
	// Register must wire s.core — service needs it for config access
	assert.NotNil(t, prep.core, "Register must set prep.core")
	assert.Equal(t, c, prep.core)
}

func TestRegister_Good_AgentsConfigLoaded(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	// Register stores agents.concurrency into Core Config — verify it is present
	concurrency := core.ConfigGet[map[string]ConcurrencyLimit](c.Config(), "agents.concurrency")
	assert.NotNil(t, concurrency, "Register must store agents.concurrency in Core Config")
}

// --- OnStartup ---

func TestPrep_OnStartup_Good_CreatesPokeCh(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.Nil(t, s.pokeCh, "pokeCh should be nil before OnStartup")

	err := s.OnStartup(context.Background())
	require.NoError(t, err)

	assert.NotNil(t, s.pokeCh, "OnStartup must initialise pokeCh via StartRunner")
}

func TestPrep_OnStartup_Good_FrozenByDefault(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	require.NoError(t, s.OnStartup(context.Background()))
	assert.True(t, s.frozen, "queue must be frozen after OnStartup without CORE_AGENT_DISPATCH=1")
}

func TestPrep_OnStartup_Good_NoError(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	err := s.OnStartup(context.Background())
	assert.NoError(t, err)
}

// --- OnShutdown ---

func TestPrep_OnShutdown_Good_FreezesQueue(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	s := &PrepSubsystem{frozen: false}
	err := s.OnShutdown(context.Background())
	require.NoError(t, err)
	assert.True(t, s.frozen, "OnShutdown must set frozen=true")
}

func TestPrep_OnShutdown_Good_AlreadyFrozen(t *testing.T) {
	// Calling OnShutdown twice must be idempotent
	s := &PrepSubsystem{frozen: true}
	err := s.OnShutdown(context.Background())
	require.NoError(t, err)
	assert.True(t, s.frozen)
}

func TestPrep_OnShutdown_Good_NoError(t *testing.T) {
	s := &PrepSubsystem{}
	assert.NoError(t, s.OnShutdown(context.Background()))
}

func TestPrep_OnShutdown_Ugly_NilCore(t *testing.T) {
	// OnShutdown must not panic even if s.core is nil
	s := &PrepSubsystem{core: nil, frozen: false}
	assert.NotPanics(t, func() {
		_ = s.OnShutdown(context.Background())
	})
	assert.True(t, s.frozen)
}
