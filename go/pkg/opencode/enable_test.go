// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestEnable_IsEnabled_Bad_DefaultsFalse — a fresh install has no persisted
// flag, so IsEnabled defaults to false (a fresh install must not auto-spawn a
// container).
func TestEnable_IsEnabled_Bad_DefaultsFalse(t *testing.T) {
	svc := newTestService(t)
	core.AssertFalse(t, svc.IsEnabled())
}

// TestEnable_setEnabled_Good_RoundTrips — setEnabled persists the flag and
// IsEnabled reflects it both ways.
func TestEnable_setEnabled_Good_RoundTrips(t *testing.T) {
	svc := newTestService(t)

	core.AssertTrue(t, svc.setEnabled(true).OK)
	core.AssertTrue(t, svc.IsEnabled())

	core.AssertTrue(t, svc.setEnabled(false).OK)
	core.AssertFalse(t, svc.IsEnabled())
}
