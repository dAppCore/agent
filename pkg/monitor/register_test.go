// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"testing"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Register_Good_ReturnsSubsystem(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	assert.True(t, service.OK)
	svc, ok := service.Value.(*Subsystem)
	assert.True(t, ok)
	assert.NotNil(t, svc)
}

func TestRegister_Register_Good_RegistersServiceName(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	assert.Contains(t, c.Services(), "monitor")
}

func TestRegister_Register_Good_WiresServiceRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	require.True(t, service.OK)
	svc, _ := service.Value.(*Subsystem)
	assert.NotNil(t, svc.ServiceRuntime)
	assert.Equal(t, c, svc.Core())
}

func TestRegister_Register_Good_TracksStartedIPC(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	require.True(t, service.OK)
	svc, ok := service.Value.(*Subsystem)
	require.True(t, ok)

	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "ws-reg"})

	unlock := svc.monitorLock()
	defer unlock()
	assert.True(t, svc.seenRunning["ws-reg"])
}

func TestRegister_Register_Good_TracksCompletedIPC(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	require.True(t, service.OK)
	svc, ok := service.Value.(*Subsystem)
	require.True(t, ok)

	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-done", Status: "completed"})

	unlock := svc.monitorLock()
	defer unlock()
	assert.True(t, svc.seenCompleted["ws-done"])
}
