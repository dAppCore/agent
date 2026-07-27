// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPaths_SetWorkspaceRootOverride_Good_AbsoluteAndRelative — absolute roots
// are used as-is; relative roots resolve against ~/Lethean.
func TestPaths_SetWorkspaceRootOverride_Good_AbsoluteAndRelative(t *testing.T) {
	t.Cleanup(func() { SetWorkspaceRootOverride("") })
	SetWorkspaceRootOverride("/abs/ws")
	core.AssertEqual(t, "/abs/ws", WorkspaceRoot())
	SetWorkspaceRootOverride("relws")
	core.AssertEqual(t, core.JoinPath(LetheanHome(), "relws"), WorkspaceRoot())
}

// TestPaths_DirHelpers_Good — the runtime dir helpers sit under ~/Lethean.
func TestPaths_DirHelpers_Good(t *testing.T) {
	core.AssertEqual(t, core.JoinPath(LetheanHome(), "log"), LogDir())
	core.AssertEqual(t, core.JoinPath(LetheanHome(), "conf"), ConfDir())
	core.AssertEqual(t, core.JoinPath(LetheanHome(), "data"), DataDir())
	core.AssertEqual(t, core.JoinPath(ConfDir(), "agents.yaml"), AgentsConfigPath())
}
