// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// A failed process Result carries a *core.Err in .Value (not a string).
// pushAndMerge used r.Value.(string), which panicked the whole binary when a
// git push / gh merge failed inside the OnStartup PR-manage loop. Exercise the
// failure branch via a fake process.run and assert it returns an error rather
// than panicking.
func TestReviewQueue_PushAndMerge_Bad_FailedResultNoPanic(t *testing.T) {
	c := core.New()
	c.Action("process.run", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{OK: false, Value: core.E("process.run", "boom", nil)}
	})
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	var err error
	core.AssertNotPanics(t, func() {
		err = pushAndMerge(s, context.Background(), "/repo", "go-io")
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "push failed")
}
