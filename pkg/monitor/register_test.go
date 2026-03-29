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
	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
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
	svc, _ := core.ServiceFor[*Subsystem](c, "monitor")
	assert.NotNil(t, svc.ServiceRuntime)
	assert.Equal(t, c, svc.Core())
}

func TestRegister_Register_Good_TracksStartedIPC(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "ws-reg"})

	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.True(t, svc.seenRunning["ws-reg"])
}

func TestRegister_Register_Good_TracksCompletedIPC(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-done", Status: "completed"})

	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.True(t, svc.seenCompleted["ws-done"])
}
