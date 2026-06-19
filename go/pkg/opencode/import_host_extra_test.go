// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestImportHost_readHostAuthJSON_GoodBad — a missing auth.json yields an empty
// map; a present one is parsed into the provider map.
func TestImportHost_readHostAuthJSON_GoodBad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	core.AssertEqual(t, 0, len(readHostAuthJSON()))

	dir := core.PathJoin(home, ".local/share/opencode")
	core.AssertTrue(t, core.MkdirAll(dir, 0o755).OK)
	core.AssertTrue(t, core.WriteFile(core.PathJoin(dir, "auth.json"), []byte(`{"anthropic":{"type":"api"}}`), 0o600).OK)
	got := readHostAuthJSON()
	core.AssertEqual(t, 1, len(got))
}
