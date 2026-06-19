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

// TestImportHost_persistProjects_Empty — an empty project array writes nothing
// and returns a zero count.
func TestImportHost_persistProjects_Empty(t *testing.T) {
	c := core.New(core.WithOption("name", "opencode-test"))
	core.AssertEqual(t, 0, persistProjects(c, []any{}, core.Now()))
}

// TestImportHost_stringFrom_Good — extract a string value; non-string or
// missing keys yield "".
func TestImportHost_stringFrom_Good(t *testing.T) {
	core.AssertEqual(t, "v", stringFrom(map[string]any{"k": "v"}, "k"))
	core.AssertEqual(t, "", stringFrom(map[string]any{"k": 1}, "k"))
	core.AssertEqual(t, "", stringFrom(map[string]any{}, "missing"))
}

// TestImportHost_projectNameFrom_Good — empty/"/" worktree falls back to the
// source id; a real path yields its basename.
func TestImportHost_projectNameFrom_Good(t *testing.T) {
	core.AssertEqual(t, "fb", projectNameFrom("", "fb"))
	core.AssertEqual(t, "fb", projectNameFrom("/", "fb"))
	core.AssertEqual(t, "repo", projectNameFrom("/home/user/repo", "fb"))
}
