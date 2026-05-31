// SPDX-Licence-Identifier: EUPL-1.2

// sigkeys.go — the minimal opencode-local slice of the desktop
// marketplace signing substrate.
//
// opencode's sigverify.go applies the require_signature policy to an
// upgrade image: it parses a base64 ed25519 public key, verifies a
// detached signature over the canonical signing bytes, and consults a
// trusted-publishers store whose on-disk shape mirrors the
// marketplace trusted-keys file. It uses ONLY:
//
//   - ParsePublicKey  — decode a base64 raw ed25519 public key.
//   - Verify          — verify a detached signature under that key.
//   - TrustedKeysFile / TrustedKey — the on-disk store shape.
//
// The desktop marketplace package carries the full bundle-manifest
// signing pipeline (CBOR canonicalisation, the trusted-keys mutation
// store, audit emission). opencode runs in a sandbox, signs nothing,
// and must not carry the audit dependency, so this file ports only
// the verify-side primitives — same crypto/ed25519 semantics, no
// store-mutation machinery, no audit.

package sigkeys

import (
	"crypto/ed25519"

	core "dappco.re/go"
)

const (
	verifyOp      = "opencode.sigkeys.Verify"
	parsePubKeyOp = "opencode.sigkeys.ParsePublicKey"

	// sigCorruptReason is emitted when a signature is structurally
	// malformed (wrong length / encoding) — distinct from
	// sigInvalidReason so the caller can distinguish "the bytes were
	// malformed" from "the bytes parsed but did not verify".
	sigCorruptReason = "sig.corrupt"

	// sigInvalidReason is emitted when a signature parses cleanly but
	// does not verify under the supplied key.
	sigInvalidReason = "sig.invalid"
)

// TrustedKey is one entry in the trusted-publishers store.
//
// Name is the human-readable priority alias. KeyID is the SHA256
// fingerprint used to select the verifying key. Pubkey is the
// base64-encoded raw ed25519 public key (32 bytes pre-encoding).
type TrustedKey struct {
	Name           string `json:"name"`
	KeyID          string `json:"keyid"`
	Pubkey         string `json:"pubkey"`
	AddedAt        string `json:"added_at"`
	AddedByAccount string `json:"added_by_account"`
}

// TrustedKeysFile is the on-disk shape at
// ~/Lethean/conf/opencode/trusted_publishers.json.
type TrustedKeysFile struct {
	Keys []TrustedKey `json:"keys"`
}

// Verify reports whether sig is a valid signature of canonical under
// pubkey. Distinguishes corrupt-signature (wrong length / encoding)
// from invalid-signature (parses but mismatches) via the returned
// reason code.
//
// Returns Ok(nil) on verify success. On failure, Result.Error()
// contains either sig.corrupt or sig.invalid as a stable prefix.
//
// Usage example:
//
//	r := sigkeys.Verify(pub, canonical, sig)
//	if !r.OK { /* r.Error() starts with "sig.corrupt: " or "sig.invalid: " */ }
func Verify(pubkey ed25519.PublicKey, canonical, sig []byte) core.Result {
	if len(pubkey) != ed25519.PublicKeySize {
		return core.Fail(core.E(verifyOp,
			sigCorruptReason+": public key size "+
				core.Sprintf("%d", len(pubkey))+
				" (want "+core.Sprintf("%d", ed25519.PublicKeySize)+")", nil))
	}
	if len(sig) != ed25519.SignatureSize {
		return core.Fail(core.E(verifyOp,
			sigCorruptReason+": signature size "+
				core.Sprintf("%d", len(sig))+
				" (want "+core.Sprintf("%d", ed25519.SignatureSize)+")", nil))
	}
	if !ed25519.Verify(pubkey, canonical, sig) {
		return core.Fail(core.E(verifyOp,
			sigInvalidReason+": signature does not verify under key", nil))
	}
	return core.Ok(nil)
}

// ParsePublicKey decodes a base64-encoded raw ed25519 public key. The
// store carries base64 pubkey bytes directly (no PEM armouring) which
// forecloses the PEM-parser bugs that have historically been a source
// of signature-bypass CVEs.
//
// Usage example:
//
//	r := sigkeys.ParsePublicKey("MCowBQYDK2VwAyEA...")
//	if r.OK { pub := r.Value.(ed25519.PublicKey) }
func ParsePublicKey(b64 string) core.Result {
	r := core.Base64Decode(core.Trim(b64))
	if !r.OK {
		return core.Fail(core.E(parsePubKeyOp,
			"public key not valid base64", nil))
	}
	raw, _ := r.Value.([]byte)
	if len(raw) != ed25519.PublicKeySize {
		return core.Fail(core.E(parsePubKeyOp,
			core.Sprintf("public key length %d (want %d)",
				len(raw), ed25519.PublicKeySize), nil))
	}
	return core.Ok(ed25519.PublicKey(raw))
}
