// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestImportHost_unixMillis — the unix-ms converter handles float64, zero, and
// non-numeric inputs.
func TestImportHost_unixMillis(t *testing.T) {
	core.AssertTrue(t, unixMillis(float64(0)).IsZero())
	core.AssertFalse(t, unixMillis(float64(1700000000000)).IsZero())
	core.AssertTrue(t, unixMillis("nope").IsZero())
}

// TestEnable_readEnabledFlag — a fresh store reports the enabled flag as absent.
func TestEnable_readEnabledFlag(t *testing.T) {
	svc := newTestService(t)
	raw, ok := svc.readEnabledFlag()
	core.AssertFalse(t, ok)
	core.AssertEqual(t, "", raw)
}
