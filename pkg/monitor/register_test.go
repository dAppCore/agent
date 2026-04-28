// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

func TestRegister_Register_Good_ReturnsSubsystem(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	core.AssertTrue(t, service.OK)
	svc, ok := service.Value.(*Subsystem)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, svc)
}

func TestRegister_Register_Good_RegistersServiceName(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	core.AssertContains(t, c.Services(), "monitor")
}

func TestRegister_Register_Good_WiresServiceRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	core.RequireTrue(t, service.OK)
	svc, _ := service.Value.(*Subsystem)
	core.AssertNotNil(t, svc.ServiceRuntime)
	core.AssertEqual(t, c, svc.Core())
}

func TestRegister_Register_Good_TracksStartedIPC(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	core.RequireTrue(t, service.OK)
	svc, ok := service.Value.(*Subsystem)
	core.RequireTrue(t, ok)

	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "ws-reg"})

	unlock := svc.monitorLock()
	defer unlock()
	core.AssertTrue(t, svc.seenRunning["ws-reg"])
}

func TestRegister_Register_Good_TracksCompletedIPC(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	core.RequireTrue(t, service.OK)
	svc, ok := service.Value.(*Subsystem)
	core.RequireTrue(t, ok)

	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-done", Status: "completed"})

	unlock := svc.monitorLock()
	defer unlock()
	core.AssertTrue(t, svc.seenCompleted["ws-done"])
}
