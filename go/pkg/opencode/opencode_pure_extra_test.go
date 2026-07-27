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

// TestOpencode_image_CustomOverridesDefault — image() returns the
// operator-supplied Options.Image verbatim when set (the non-empty arm);
// an unset Image falls back to defaultImage. Covers both branches of the
// per-spawn image resolver.
func TestOpencode_image_CustomOverridesDefault(t *testing.T) {
	c := core.New(core.WithOption("name", "opencode-test"))
	custom := NewService(Options{Image: "forge.lthn.sh/lthn/dev:pinned"})(c)
	core.AssertTrue(t, custom.OK)
	csvc, _ := custom.Value.(*Service)
	core.AssertEqual(t, "forge.lthn.sh/lthn/dev:pinned", csvc.image())

	d := core.New(core.WithOption("name", "opencode-test"))
	def := NewService(Options{})(d)
	core.AssertTrue(t, def.OK)
	dsvc, _ := def.Value.(*Service)
	core.AssertEqual(t, defaultImage, dsvc.image())
}
