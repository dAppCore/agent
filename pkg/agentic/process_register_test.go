// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessRegister_ProcessRegister_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	result := ProcessRegister(c)
	assert.True(t, result.OK, "ProcessRegister should succeed with a real Core instance")
	assert.True(t, c.Process().Exists(), "ProcessRegister should register the process service")
}

func TestProcessRegister_ProcessRegister_Bad_NilCore(t *testing.T) {
	result := ProcessRegister(nil)
	assert.False(t, result.OK)
}

func TestProcessRegister_ProcessRegister_Ugly_DoubleRegister(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	r1 := ProcessRegister(c)
	assert.True(t, r1.OK)

	r2 := ProcessRegister(c)
	assert.True(t, r2.OK, "second ProcessRegister call should not fail")
}

func TestProcessRegister_ProcessRegister_Ugly_PreRegisteredService(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	factory := process.NewService(process.Options{})
	instance, err := factory(c)
	require.NoError(t, err)

	service, ok := instance.(*process.Service)
	require.True(t, ok)
	require.True(t, c.RegisterService("process", service).OK)

	result := ProcessRegister(c)
	require.True(t, result.OK)
	assert.True(t, c.Action("process.run").Exists(), "existing process service should still register actions")
	assert.True(t, c.Action("process.start").Exists(), "existing process service should still register actions")
	assert.True(t, c.Action("process.kill").Exists(), "existing process service should still register actions")
}

func TestProcessRegister_HandleRun_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	require.True(t, ProcessRegister(c).OK)

	r := c.Action("process.run").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "echo"},
		core.Option{Key: "args", Value: []string{"ok"}},
	))
	require.True(t, r.OK)
	assert.Equal(t, "ok", core.Trim(r.Value.(string)))
}

func TestProcessRegister_HandleKill_Bad_MissingID(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	require.True(t, ProcessRegister(c).OK)

	r := c.Action("process.kill").Run(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestProcessRegister_HandleStart_Ugly_StartAndKill(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New()
	require.True(t, ProcessRegister(c).OK)

	r := c.Action("process.start").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	require.True(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	require.True(t, ok)
	require.NotEmpty(t, proc.ID)

	defer proc.Kill()

	kill := c.Action("process.kill").Run(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: proc.ID},
	))
	assert.True(t, kill.OK)

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

func TestProcessRegister_OverrideService_Good_ServiceStartupPreservesAgentHandlers(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	c := core.New(core.WithService(ProcessRegister))
	require.True(t, c.ServiceStartup(context.Background(), nil).OK)

	r := c.Action("process.start").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	require.True(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	require.True(t, ok, "agent-side process.start must still return *process.Process after ServiceStartup")
	require.NotEmpty(t, proc.ID)

	defer proc.Kill()
}

func TestProcessRegister_OverrideService_Bad_NilHandlers(t *testing.T) {
	svc := &processOverrideService{}
	result := svc.OnStartup(context.Background())
	assert.True(t, result.OK, "OnStartup with nil handlers should succeed without panicking")
}

func TestProcessRegister_OverrideService_Ugly_NilCore(t *testing.T) {
	svc := &processOverrideService{handlers: &processActionHandlers{}, core: nil}
	result := svc.OnStartup(context.Background())
	assert.True(t, result.OK, "OnStartup with nil core should no-op without panic")
}
