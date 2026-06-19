// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommandsPlan_cmdPlanShow_Bad_RequiresSlug — plan show without a slug
// prints usage and returns a slug-required error.
func TestCommandsPlan_cmdPlanShow_Bad_RequiresSlug(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanShow(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent plan show <slug>")
}

// TestCommandsPlan_cmdPlanList_Good_EmptyStore — plan list against an empty
// workspace reports no plans and succeeds.
func TestCommandsPlan_cmdPlanList_Good_EmptyStore(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanList(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "no plans")
}
