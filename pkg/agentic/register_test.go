// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- Register ---

func TestRegister_ServiceRegistered_Good_Case(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_BRAIN_URL", "")

	c := core.New(core.WithService(Register))
	core.AssertNotNil(t, c)

	// Service auto-registered under the last segment of the package path: "agentic"
	service := c.Service("agentic")
	core.RequireTrue(t, service.OK)
	prep, ok := service.Value.(*PrepSubsystem)
	core.RequireTrue(t, ok, "PrepSubsystem must be registered as \"agentic\"")
	core.AssertNotNil(t, prep)
}

func TestRegister_CoreWired_Good_Case(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	service := c.Service("agentic")
	core.RequireTrue(t, service.OK)
	prep, ok := service.Value.(*PrepSubsystem)
	core.RequireTrue(t, ok)
	// Register must wire ServiceRuntime — service needs it for Core access
	core.AssertNotNil(t, prep.ServiceRuntime, "Register must set ServiceRuntime")
	core.AssertEqual(t, c, prep.Core())
}

func TestRegister_AgentsConfig_Good_Case(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_URL", "")

	c := core.New(core.WithService(Register))

	// Register stores agents.concurrency into Core Config — verify it is present
	concurrency := core.ConfigGet[map[string]ConcurrencyLimit](c.Config(), "agents.concurrency")
	core.AssertNotNil(t, concurrency, "Register must store agents.concurrency in Core Config")
}

func TestRegister_Register_Good(t *testing.T) {
	manager := &ProviderManager{}
	provider := newContentProvider("custom", "gpt-5.4", true, nil)
	manager.Register(provider)
	core.AssertContains(t, manager.Names(), "custom")
	core.AssertEqual(t, provider, manager.DefaultProvider())
}

func TestRegister_Register_Bad(t *testing.T) {
	manager := &ProviderManager{}
	manager.Register(nil)
	core.AssertNil(t, manager.Names())
	core.AssertNil(t, manager.DefaultProvider())
}

func TestRegister_Register_Ugly(t *testing.T) {
	manager := &ProviderManager{}
	first := newContentProvider("custom", "v1", true, nil)
	second := newContentProvider("custom", "v2", true, nil)
	manager.Register(first)
	manager.Register(second)
	provider, ok := manager.Provider("custom")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "v2", provider.DefaultModel())
}
