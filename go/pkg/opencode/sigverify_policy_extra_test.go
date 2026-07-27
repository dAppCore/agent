// SPDX-Licence-Identifier: EUPL-1.2

// verifySignatureForUpgrade arm coverage, driven directly (the function
// takes *Service + UpgradeInput + canonical bytes, so it unit-tests
// without the docker-pull side-effect UpgradeWithConsent carries). The
// existing sigverify_test.go drives the verify decision through
// UpgradeWithConsent's outer gates; this file pins three arms that path
// doesn't reach cleanly:
//
//   - policy off + no signature → silent accept
//   - a malformed public key → reject (sig.corrupt)
//   - a malformed trusted_publishers.json → reject (load propagates)
//
// Reuses newServiceWithPolicy + writeTrustedPublishers from the sibling
// sigverify tests. No docker, no network.

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestVerifySignature_PolicyOffNoSig_Accept — require_signature=false and
// no signature supplied is the legacy/bootstrap accept: Ok(nil), no audit.
func TestVerifySignature_PolicyOffNoSig_Accept(t *testing.T) {
	svc := newServiceWithPolicy(t, false)
	canonical, ok := canonicalSigningBytes("sha256:"+repeat64('a'), "v1", "rel-1")
	core.AssertTrue(t, ok)

	r := verifySignatureForUpgrade(svc, UpgradeInput{ImageDigest: "sha256:" + repeat64('a')}, canonical)
	core.AssertTrue(t, r.OK)
}

// TestVerifySignature_MalformedPublicKey_Reject — a signature + a public
// key that is not valid base64-raw-ed25519 rejects with sig.corrupt at the
// ParsePublicKey arm, before any trust-store lookup.
func TestVerifySignature_MalformedPublicKey_Reject(t *testing.T) {
	svc := newServiceWithPolicy(t, true)
	canonical, ok := canonicalSigningBytes("sha256:"+repeat64('b'), "v1", "rel-2")
	core.AssertTrue(t, ok)

	in := UpgradeInput{
		ImageDigest:     "sha256:" + repeat64('b'),
		SignatureBytes:  []byte("not-a-real-signature"),
		PublicKeyBase64: []byte("@@@ not base64 @@@"),
	}
	r := verifySignatureForUpgrade(svc, in, canonical)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), sigReasonCorrupt)
}

// TestVerifySignature_TrustStoreLoadFails_Reject — a well-formed public key
// but a malformed trusted_publishers.json makes loadTrustedPublishers fail,
// and verifySignatureForUpgrade propagates that failure (the operator's
// trust store is unreadable → fail closed, never silently accept).
func TestVerifySignature_TrustStoreLoadFails_Reject(t *testing.T) {
	// writeTrustedPublishers sets a temp HOME + writes the (malformed) store.
	writeTrustedPublishers(t, `{"keys":[ truncated`)

	svc := newServiceWithPolicy(t, true)
	canonical, ok := canonicalSigningBytes("sha256:"+repeat64('c'), "v1", "rel-3")
	core.AssertTrue(t, ok)

	// A syntactically valid base64 ed25519-length key (32 bytes → 44 b64
	// chars) so ParsePublicKey succeeds and we reach the trust-store load.
	in := UpgradeInput{
		ImageDigest:     "sha256:" + repeat64('c'),
		SignatureBytes:  []byte("sig-bytes"),
		PublicKeyBase64: []byte(core.Base64Encode(make([]byte, 32))),
	}
	r := verifySignatureForUpgrade(svc, in, canonical)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "parse failed")
}

// repeat64 builds a 64-char digest hex tail for sha256:<64hex> shapes.
func repeat64(c byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = c
	}
	return string(b)
}
