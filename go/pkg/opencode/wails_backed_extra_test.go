// SPDX-Licence-Identifier: EUPL-1.2

// Success-delegation coverage for the Wails binding wrappers. The existing
// wails_extra_test.go proves the nil-guard arms; wails_provider_test.go
// proves the masking helper in isolation (a copy of the loop). Neither
// drives the *method bodies* of the bound-service success path. This file
// does, on the kv-backed + ORM-backed newBackedService, asserting the real
// side-effect each wrapper produces (round-trip read-back, file content,
// redacted-but-present provider view) rather than "OK".
//
// No docker / no container: profile CRUD is KV, the import providers are
// ORM rows persisted via persistProviders, the host-config merge writes a
// file under the test's temp HOME. The Start-spawning wrappers (WStart /
// WEnable) are NOT exercised on their success path — they would create a
// real container.

package opencode

import (
	"encoding/json"
	"testing"

	core "dappco.re/go"
)

// TestWSaveProfile_Good_RoundTrips — WSaveProfile on a bound service
// persists, and WGetProfile reads the same profile back. Asserts the
// stored Model survives the round-trip, proving the wrapper reached
// Service.SaveProfile (not just returned OK).
func TestWSaveProfile_Good_RoundTrips(t *testing.T) {
	svc := newBackedService(t)
	w := NewWailsService(svc)

	saved := Profile{Name: "round-trip", Model: "anthropic/claude-sonnet-4-5"}
	core.AssertTrue(t, w.WSaveProfile(saved).OK)

	got := w.WGetProfile("round-trip")
	core.AssertTrue(t, got.OK)
	p, ok := got.Value.(Profile)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "round-trip", p.Name)
	core.AssertEqual(t, "anthropic/claude-sonnet-4-5", p.Model)
}

// TestWSaveProfile_Bad_SchemaInvalid — WSaveProfile delegates schema
// validation: an unknown provider id surfaces the Service failure
// (ProfileInvalidSchema) through the wrapper rather than persisting.
func TestWSaveProfile_Bad_SchemaInvalid(t *testing.T) {
	svc := newBackedService(t)
	w := NewWailsService(svc)

	bad := Profile{
		Name:     "bad",
		Provider: map[string]any{"evil": map[string]any{"npm": "@attacker/sdk"}},
	}
	r := w.WSaveProfile(bad)
	core.AssertFalse(t, r.OK)
	core.AssertEqual(t, ProfileInvalidSchema, r.Code())

	// The failed save must not have created the profile.
	core.AssertFalse(t, w.WGetProfile("bad").OK)
}

// TestWDeleteProfile_Good_RemovesRow — save then WDeleteProfile then
// confirm WGetProfile fails: the wrapper reaches Service.DeleteProfile and
// the row is gone.
func TestWDeleteProfile_Good_RemovesRow(t *testing.T) {
	svc := newBackedService(t)
	w := NewWailsService(svc)

	core.AssertTrue(t, w.WSaveProfile(Profile{Name: "ephemeral"}).OK)
	core.AssertTrue(t, w.WGetProfile("ephemeral").OK)

	core.AssertTrue(t, w.WDeleteProfile("ephemeral").OK)
	core.AssertFalse(t, w.WGetProfile("ephemeral").OK)
}

// TestWDeleteProfile_Bad_DefaultProtected — WDeleteProfile of "default"
// surfaces the protected-name failure from Service.DeleteProfile.
func TestWDeleteProfile_Bad_DefaultProtected(t *testing.T) {
	svc := newBackedService(t)
	w := NewWailsService(svc)

	r := w.WDeleteProfile("default")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "default profile")
}

// TestWMergeHostConfig_Good_WritesFile — WMergeHostConfig on a bound
// service merges the default profile's provider block into the host
// opencode.json (under temp HOME). Asserts the wrapper reached
// Service.MergeHostConfig: the result reports Created=true with the temp
// path, the in-process Bytes carry the provider block, and the file
// actually landed on disk with that block. Bytes is json:"-" (never
// wire-encoded), so the assertion reads the struct field + the file, not
// the marshalled view.
func TestWMergeHostConfig_Good_WritesFile(t *testing.T) {
	svc := newBackedService(t)
	w := NewWailsService(svc)

	r := w.WMergeHostConfig(MergeHostConfigOptions{Profile: DefaultProfile})
	core.AssertTrue(t, r.OK)

	res, ok := r.Value.(MergeHostConfigResult)
	core.AssertTrue(t, ok)
	core.AssertTrue(t, res.Created)
	core.AssertEqual(t, DefaultProfile, res.Profile)
	core.AssertContains(t, res.Path, "opencode.json")
	// The in-process pretty-printed bytes carry the merged provider block.
	core.AssertContains(t, res.Bytes, "provider")

	// The merge actually wrote the file on disk with the provider block.
	fr := core.ReadFile(res.Path)
	core.AssertTrue(t, fr.OK)
	onDisk, _ := fr.Value.([]byte)
	core.AssertContains(t, string(onDisk), "provider")
}

// TestWMergeHostConfig_Bad_UnknownProfile — WMergeHostConfig for a profile
// that does not exist surfaces the GetProfile not-found failure through the
// wrapper, writing nothing.
func TestWMergeHostConfig_Bad_UnknownProfile(t *testing.T) {
	svc := newBackedService(t)
	w := NewWailsService(svc)

	r := w.WMergeHostConfig(MergeHostConfigOptions{Profile: "no-such-profile"})
	core.AssertFalse(t, r.OK)
}

// TestWListImportedProviders_Good_RedactedViews — with two persisted
// providers (one carrying an auth key) WListImportedProviders returns
// ProviderView rows through the *method body* (not the copied loop): the
// auth-bearing row reports Present=true with a non-empty Masked, the
// keyless row reports Present=false, and the raw key never appears in the
// serialised payload.
func TestWListImportedProviders_Good_RedactedViews(t *testing.T) {
	const rawKey = "sk-ant-api03-LIVE-SECRET-DO-NOT-LEAK-4f2a"

	svc := newBackedService(t)
	w := NewWailsService(svc)

	// Persist two providers; anthropic carries an auth key via the authMap.
	providers := []any{
		map[string]any{"id": "anthropic", "name": "Anthropic"},
		map[string]any{"id": "openai", "name": "OpenAI"},
	}
	authMap := map[string]map[string]any{
		"anthropic": {"type": "apikey", "key": rawKey},
	}
	count, withAuth := persistProviders(svc.Core(), providers, authMap, core.Now())
	core.AssertEqual(t, 2, count)
	core.AssertEqual(t, 1, withAuth)

	r := w.WListImportedProviders()
	core.AssertTrue(t, r.OK)
	views, ok := r.Value.([]ProviderView)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 2, len(views))

	// Locate the two rows by provider id and assert the redaction contract.
	byID := map[string]ProviderView{}
	for _, v := range views {
		byID[v.ProviderID] = v
	}
	anth, hasAnth := byID["anthropic"]
	core.AssertTrue(t, hasAnth)
	core.AssertTrue(t, anth.Present)
	core.AssertNotEmpty(t, anth.Masked)
	core.AssertNotEqual(t, rawKey, anth.Masked)

	oai, hasOAI := byID["openai"]
	core.AssertTrue(t, hasOAI)
	core.AssertFalse(t, oai.Present)
	core.AssertEqual(t, "", oai.Masked)

	// Defence-in-depth: the serialised bridge payload must not leak the key.
	b, err := json.Marshal(views)
	if err != nil {
		t.Fatalf("marshal views: %v", err)
	}
	if contains(string(b), rawKey) {
		t.Errorf("WListImportedProviders payload leaked the raw AuthKey: %s", string(b))
	}
}
