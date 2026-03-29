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
