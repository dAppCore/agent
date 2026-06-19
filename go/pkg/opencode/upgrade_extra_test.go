// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestUpgrade_parsePullDigest_GoodBad — the digest is pulled from the "Digest:"
// line; absent → "".
func TestUpgrade_parsePullDigest_GoodBad(t *testing.T) {
	core.AssertEqual(t, "sha256:abc123",
		parsePullDigest("Pulling...\nDigest: sha256:abc123\nStatus: done"))
	core.AssertEqual(t, "", parsePullDigest("no digest here"))
}

// TestUpgrade_validSHA256Digest_GoodBad — canonical sha256:64-hex passes;
// wrong length / algorithm / empty fail.
func TestUpgrade_validSHA256Digest_GoodBad(t *testing.T) {
	core.AssertTrue(t, validSHA256Digest("sha256:ca59eb28d5ea6a1f50c45a1f1df5c1a9286343e41b389fe89fb4ffac96dbeb84"))
	core.AssertFalse(t, validSHA256Digest("sha256:CA59EB28"))
	core.AssertFalse(t, validSHA256Digest("md5:abcd"))
	core.AssertFalse(t, validSHA256Digest(""))
}
