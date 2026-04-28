// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/process"
)

func TestProcessRegister_ProcessRegister_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	result := ProcessRegister(c)
	core.AssertTrue(t, result.OK, "ProcessRegister should succeed with a real Core instance")
	core.AssertTrue(t, c.Process().Exists(), "ProcessRegister should register the process service")
}

func TestProcessRegisterNilCore_ProcessRegister_Bad(t *testing.T) {
	result := ProcessRegister(nil)
	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "core is required")
}

func TestProcessRegisterDoubleRegister_ProcessRegister_Ugly(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	r1 := ProcessRegister(c)
	core.AssertTrue(t, r1.OK)

	r2 := ProcessRegister(c)
	core.AssertTrue(t, r2.OK, "second ProcessRegister call should not fail")
}

func TestProcessRegister_ProcessRegister_Ugly_PreRegisteredService(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	factory := process.NewService(process.Options{})
	instance, err := factory(c)
	core.RequireNoError(t, err)

	service, ok := instance.(*process.Service)
	core.RequireTrue(t, ok)
	core.RequireTrue(t, c.RegisterService("process", service).OK)

	result := ProcessRegister(c)
	core.RequireTrue(t, result.OK)
	core.AssertTrue(t, c.Action("process.run").Exists(), "existing process service should still register actions")
	core.AssertTrue(t, c.Action("process.start").Exists(), "existing process service should still register actions")
	core.AssertTrue(t, c.Action("process.kill").Exists(), "existing process service should still register actions")
}

func TestProcessRegister_HandleRun_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	core.RequireTrue(t, ProcessRegister(c).OK)

	r := c.Action("process.run").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "echo"},
		core.Option{Key: "args", Value: []string{"ok"}},
	))
	core.RequireTrue(t, r.OK)
	core.AssertEqual(t, "ok", core.Trim(r.Value.(string)))
}

func TestProcessRegister_HandleKill_Bad_MissingID(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	core.RequireTrue(t, ProcessRegister(c).OK)

	r := c.Action("process.kill").Run(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestProcessRegister_HandleStart_Ugly_StartAndKill(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	core.RequireTrue(t, ProcessRegister(c).OK)

	r := c.Action("process.start").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	core.RequireTrue(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	core.RequireTrue(t, ok)
	core.RequireNotEmpty(t, proc.ID)

	defer proc.Kill()

	kill := c.Action("process.kill").Run(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: proc.ID},
	))
	core.AssertTrue(t, kill.OK)

	select {
	case <-proc.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("process.kill did not stop the managed process")
	}
}

// ProcessOverrideService guards the RFC §7 dispatch contract: go-process's own
// OnStartup registers string-ID variants of `process.*` which would break the
// agent's pid/queue helpers. The override service reapplies agent handlers
// after ServiceStartup so the custom `*process.Process`-returning handlers win.

func TestProcessRegisterHandlers_OverrideService_OnStartup_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New(core.WithService(ProcessRegister))
	core.RequireTrue(t, c.ServiceStartup(context.Background(), nil).OK)

	r := c.Action("process.start").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	core.RequireTrue(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	core.RequireTrue(t, ok, "agent-side process.start must still return *process.Process after ServiceStartup")
	core.RequireNotEmpty(t, proc.ID)

	defer proc.Kill()
}

func TestNilHandlers_OverrideService_OnStartup_Bad(t *testing.T) {
	svc := &processOverrideService{}
	result := svc.OnStartup(context.Background())
	core.AssertTrue(t, result.OK, "OnStartup with nil handlers should succeed without panicking")
}

func TestNilCore_OverrideService_OnStartup_Ugly(t *testing.T) {
	svc := &processOverrideService{handlers: &processActionHandlers{}, core: nil}
	result := svc.OnStartup(context.Background())
	core.AssertTrue(t, result.OK, "OnStartup with nil core should no-op without panic")
}
