// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommands_cmdPersonas_Good_HumanAndJSON — personas lists the persona
// cards in both the human and --json lanes; both succeed.
func TestCommands_cmdPersonas_Good_HumanAndJSON(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var rh, rj core.Result
	out := captureStdout(t, func() {
		rh = s.cmdPersonas(core.NewOptions())
		rj = s.cmdPersonas(core.NewOptions(core.Option{Key: "json", Value: true}))
	})
	core.AssertTrue(t, rh.OK)
	core.AssertTrue(t, rj.OK)
	core.AssertContains(t, out, "personas:")
}

// TestCommands_cmdTasks_Good_HumanAndJSON — tasks lists the task-template
// cards in both the human and --json lanes; both succeed.
func TestCommands_cmdTasks_Good_HumanAndJSON(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var rh, rj core.Result
	out := captureStdout(t, func() {
		rh = s.cmdTasks(core.NewOptions())
		rj = s.cmdTasks(core.NewOptions(core.Option{Key: "json", Value: true}))
	})
	core.AssertTrue(t, rh.OK)
	core.AssertTrue(t, rj.OK)
	core.AssertContains(t, out, "tasks:")
}
