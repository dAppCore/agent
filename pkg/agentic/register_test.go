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

func TestRegister_ServiceRegistered_Good(t *testing.T) {
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

func TestRegister_CoreWired_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	prep, ok := core.ServiceFor[*PrepSubsystem](c, "agentic")
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

// --- ProcessRegister ---

func TestRegister_ProcessRegister_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	result := ProcessRegister(c)
	assert.True(t, result.OK, "ProcessRegister should succeed with a real Core instance")
	assert.True(t, c.Process().Exists(), "ProcessRegister should register the process service")
}

func TestRegister_ProcessRegister_Bad(t *testing.T) {
	// nil Core — the process.NewService factory tolerates nil Core, returns a result
	result := ProcessRegister(nil)
	// Either OK (service created without Core) or not OK (error) — must not panic
	_ = result
}

func TestRegister_ProcessRegister_Ugly(t *testing.T) {
	// Call twice with same Core — second call should still succeed
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	r1 := ProcessRegister(c)
	assert.True(t, r1.OK)

	r2 := ProcessRegister(c)
	assert.True(t, r2.OK, "second ProcessRegister call should not fail")
}

// --- OnStartup ---

func TestPrep_OnStartup_Good_CreatesPokeCh(t *testing.T) {
	// StartRunner is now a no-op — pokeCh is no longer initialised by OnStartup.
	// Verify OnStartup succeeds and pokeCh remains nil.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.Nil(t, s.pokeCh, "pokeCh should be nil before OnStartup")

	r := s.OnStartup(context.Background())
	assert.True(t, r.OK)

	assert.Nil(t, s.pokeCh, "pokeCh should remain nil — queue drain is owned by pkg/runner")
}

func TestPrep_OnStartup_Good_FrozenByDefault(t *testing.T) {
	// Frozen state is now owned by pkg/runner.Service, not agentic.
	// Verify OnStartup succeeds without asserting frozen state.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.True(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Good_NoError(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.True(t, s.OnStartup(context.Background()).OK)
}

// --- OnShutdown ---

func TestPrep_OnShutdown_Good_FreezesQueue(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: false}
	r := s.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, s.frozen, "OnShutdown must set frozen=true")
}

func TestPrep_OnShutdown_Good_AlreadyFrozen(t *testing.T) {
	// Calling OnShutdown twice must be idempotent
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: true}
	r := s.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, s.frozen)
}

func TestPrep_OnShutdown_Good_NoError(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.True(t, s.OnShutdown(context.Background()).OK)
}

func TestPrep_OnShutdown_Ugly_NilCore(t *testing.T) {
	// OnShutdown must not panic even if s.core is nil
	s := &PrepSubsystem{ServiceRuntime: nil, frozen: false}
	assert.NotPanics(t, func() {
		_ = s.OnShutdown(context.Background())
	})
	assert.True(t, s.frozen)
}
