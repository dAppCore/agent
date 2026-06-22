// SPDX-Licence-Identifier: EUPL-1.2

// Coverage for the readEnabledFlag future-arc helper and the IsEnabled
// missing-key default. The existing enable_extra_test.go covers Enable /
// Disable (which spawn / stop), but readEnabledFlag is a standalone lookup
// the lifecycle path never calls, and IsEnabled's no-key default is only
// implicitly hit. Both run on the kv-backed newTestService — pure KV, no
// container.

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestEnable_readEnabledFlag_PresentTrueFalse_Good — after setEnabled the
// raw flag is readable with present=true and the stored string ("true" /
// "false"). Pins the success-return arm of readEnabledFlag (the helper
// that distinguishes "explicitly disabled" from "never set").
func TestEnable_readEnabledFlag_PresentTrueFalse_Good(t *testing.T) {
	svc := newTestService(t)

	core.AssertTrue(t, svc.setEnabled(true).OK)
	raw, present := svc.readEnabledFlag()
	core.AssertTrue(t, present)
	core.AssertEqual(t, enabledTrue, raw)

	core.AssertTrue(t, svc.setEnabled(false).OK)
	raw, present = svc.readEnabledFlag()
	core.AssertTrue(t, present)
	core.AssertEqual(t, enabledFalse, raw)
}

// TestEnable_readEnabledFlag_MissingKey_NotPresent — on a fresh store the
// key is absent: readEnabledFlag returns ("", false) (the NotFound arm)
// and IsEnabled defaults to false (no auto-spawn on a clean install).
func TestEnable_readEnabledFlag_MissingKey_NotPresent(t *testing.T) {
	svc := newTestService(t)

	raw, present := svc.readEnabledFlag()
	core.AssertFalse(t, present)
	core.AssertEqual(t, "", raw)

	core.AssertFalse(t, svc.IsEnabled())
}
