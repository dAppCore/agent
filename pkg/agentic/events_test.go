// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestEvents_EmitEvent_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	core.AssertNotPanics(t, func() {
		emitStartEvent("codex", "ws-1")
	})
}

func TestEvents_EmitEvent_Bad_NoWorkspace(t *testing.T) {
	setTestWorkspace(t, "/nonexistent")
	core.AssertNotPanics(t, func() {
		emitCompletionEvent("codex", "ws-1", "completed")
	})
}

func TestEvents_EmitEvent_Ugly_AllEmpty(t *testing.T) {
	core.AssertNotPanics(t, func() {
		emitEvent("", "", "", "")
	})
}
