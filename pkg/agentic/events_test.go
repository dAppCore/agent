// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestEvents_EmitEvent_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	assert.NotPanics(t, func() {
		emitStartEvent("codex", "ws-1")
	})
}

func TestEvents_EmitEvent_Bad_NoWorkspace(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/nonexistent")
	assert.NotPanics(t, func() {
		emitCompletionEvent("codex", "ws-1", "completed")
	})
}

func TestEvents_EmitEvent_Ugly_AllEmpty(t *testing.T) {
	assert.NotPanics(t, func() {
		emitEvent("", "", "", "")
	})
}
