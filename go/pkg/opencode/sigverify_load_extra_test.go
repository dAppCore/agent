// SPDX-Licence-Identifier: EUPL-1.2

// loadTrustedPublishers coverage — the upgrade signature trust store
// (Cerberus #22 MED-2 / Mantis #1622). The existing sigverify_test.go
// drives the verify decision via UpgradeWithConsent; this file targets the
// store loader's own reject arms directly: a malformed file, a row missing
// name/keyid, and the N1 invariant (same name + different keyid REJECTS).
// Each writes a fixture under a temp HOME so trustedPublishersPath()
// resolves to it — pure filesystem, no docker, no network.

package opencode

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/opencode/internal/sigkeys"
)

// writeTrustedPublishers points HOME at a temp dir and writes raw bytes to
// ~/Lethean/conf/opencode/trusted_publishers.json, returning the path.
func writeTrustedPublishers(t *testing.T, raw string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	path := trustedPublishersPath()
	core.AssertTrue(t, core.MkdirAll(core.PathDir(path), 0o755).OK)
	core.AssertTrue(t, core.WriteFile(path, []byte(raw), 0o600).OK)
	return path
}

// TestLoadTrustedPublishers_Good_ParsesKeys — a well-formed two-key file
// loads and the parsed rows survive (name + keyid round-trip). The clean
// accept path of the loader.
func TestLoadTrustedPublishers_Good_ParsesKeys(t *testing.T) {
	writeTrustedPublishers(t, `{"keys":[
		{"name":"release-eng","keyid":"kid-1","pubkey":"AAAA"},
		{"name":"backup-signer","keyid":"kid-2","pubkey":"BBBB"}
	]}`)

	r := loadTrustedPublishers()
	core.AssertTrue(t, r.OK)
	tf, ok := r.Value.(sigkeys.TrustedKeysFile)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 2, len(tf.Keys))
	core.AssertEqual(t, "release-eng", tf.Keys[0].Name)
	core.AssertEqual(t, "kid-2", tf.Keys[1].KeyID)
}

// TestLoadTrustedPublishers_Bad_MalformedJSON — a syntactically invalid
// file fails the parse arm rather than silently loading an empty store
// (which would weaken the require_signature policy to a no-op).
func TestLoadTrustedPublishers_Bad_MalformedJSON(t *testing.T) {
	writeTrustedPublishers(t, `{"keys":[ {"name":"x" `) // truncated

	r := loadTrustedPublishers()
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "parse failed")
}

// TestLoadTrustedPublishers_Bad_EmptyNameOrKeyID — a row missing keyid
// (or name) rejects: both fields are required so an attacker can't smuggle
// a half-formed row past the loader.
func TestLoadTrustedPublishers_Bad_EmptyNameOrKeyID(t *testing.T) {
	writeTrustedPublishers(t, `{"keys":[{"name":"release-eng","keyid":""}]}`)

	r := loadTrustedPublishers()
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "name and keyid are required")
}

// TestLoadTrustedPublishers_Bad_DuplicateNameDifferentKeyID — the N1
// invariant (DREAD v2 N1 HIGH): the same publisher name appearing twice
// with DIFFERENT keyids is a shadow-entry attack and must reject. A second
// row that re-binds "release-eng" to a new key is the canonical shape.
func TestLoadTrustedPublishers_Bad_DuplicateNameDifferentKeyID(t *testing.T) {
	writeTrustedPublishers(t, `{"keys":[
		{"name":"release-eng","keyid":"kid-legit","pubkey":"AAAA"},
		{"name":"release-eng","keyid":"kid-attacker","pubkey":"EVIL"}
	]}`)

	r := loadTrustedPublishers()
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "duplicate name with different keyid")
	core.AssertContains(t, r.Error(), "release-eng")
}

// TestLoadTrustedPublishers_Good_SameNameSameKeyID — the same name + the
// SAME keyid repeated is NOT an attack (idempotent re-list) and loads
// clean. Pins the negative space of the N1 check: only a keyid *change*
// trips it.
func TestLoadTrustedPublishers_Good_SameNameSameKeyID(t *testing.T) {
	writeTrustedPublishers(t, `{"keys":[
		{"name":"release-eng","keyid":"kid-1","pubkey":"AAAA"},
		{"name":"release-eng","keyid":"kid-1","pubkey":"AAAA"}
	]}`)

	r := loadTrustedPublishers()
	core.AssertTrue(t, r.OK)
	tf, _ := r.Value.(sigkeys.TrustedKeysFile)
	core.AssertEqual(t, 2, len(tf.Keys))
}

// TestLoadTrustedPublishers_Good_AbsentFileIsEmpty — no file at all is the
// bootstrap state, NOT an error: the loader returns an empty store and the
// caller's require_signature policy decides whether that is acceptable.
func TestLoadTrustedPublishers_Good_AbsentFileIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // nothing written under it

	r := loadTrustedPublishers()
	core.AssertTrue(t, r.OK)
	tf, ok := r.Value.(sigkeys.TrustedKeysFile)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 0, len(tf.Keys))
}
