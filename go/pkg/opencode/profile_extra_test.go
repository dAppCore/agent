// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestProfile_GetProfile_Bad_EmptyName — an empty name is rejected before the
// store is touched.
func TestProfile_GetProfile_Bad_EmptyName(t *testing.T) {
	svc := newTestService(t)
	core.AssertFalse(t, svc.GetProfile("").OK)
}

// TestProfile_GetProfile_Bad_NotFound — an unknown profile returns the
// notfound code.
func TestProfile_GetProfile_Bad_NotFound(t *testing.T) {
	svc := newTestService(t)
	r := svc.GetProfile("does-not-exist")
	core.AssertFalse(t, r.OK)
	core.AssertEqual(t, "opencode.profile.notfound", r.Code())
}

// TestProfile_SaveProfile_Bad_EmptyName — saving without a name is rejected.
func TestProfile_SaveProfile_Bad_EmptyName(t *testing.T) {
	svc := newTestService(t)
	core.AssertFalse(t, svc.SaveProfile(Profile{}).OK)
}

// TestProfile_SaveProfile_Good_RoundTrips — a saved profile reads back by name.
func TestProfile_SaveProfile_Good_RoundTrips(t *testing.T) {
	svc := newTestService(t)
	core.AssertTrue(t, svc.SaveProfile(Profile{Name: "tight-loop"}).OK)
	r := svc.GetProfile("tight-loop")
	core.AssertTrue(t, r.OK)
	got, ok := r.Value.(Profile)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "tight-loop", got.Name)
}

// TestProfile_ToOpenCodeWire_Good_StripsNameDescription — the wire form drops
// name + description (opencode-internal fields) but keeps the rest.
func TestProfile_ToOpenCodeWire_Good_StripsNameDescription(t *testing.T) {
	wire := Profile{Name: "NAMEFIELD", Description: "DESCFIELD", Model: "MODELVAL"}.ToOpenCodeWire()
	core.AssertFalse(t, core.Contains(wire, "NAMEFIELD"))
	core.AssertFalse(t, core.Contains(wire, "DESCFIELD"))
	core.AssertTrue(t, core.Contains(wire, "MODELVAL"))
}

// TestProfile_ListProfiles_Good — the seeded default plus a saved profile are
// both listed.
func TestProfile_ListProfiles_Good(t *testing.T) {
	svc := newTestService(t)
	core.AssertTrue(t, svc.SaveProfile(Profile{Name: "extra"}).OK)
	r := svc.ListProfiles()
	core.AssertTrue(t, r.OK)
	list, ok := r.Value.([]Profile)
	core.AssertTrue(t, ok)
	core.AssertTrue(t, len(list) >= 2)
}

// TestProfile_DeleteProfile_GuardsAndDelete — empty name + the default profile
// are refused; a real profile deletes and is then gone.
func TestProfile_DeleteProfile_GuardsAndDelete(t *testing.T) {
	svc := newTestService(t)
	core.AssertFalse(t, svc.DeleteProfile("").OK)
	core.AssertFalse(t, svc.DeleteProfile(DefaultProfile).OK)
	core.AssertTrue(t, svc.SaveProfile(Profile{Name: "temp"}).OK)
	core.AssertTrue(t, svc.DeleteProfile("temp").OK)
	core.AssertFalse(t, svc.GetProfile("temp").OK)
}
