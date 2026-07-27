// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// TestFetchLoopCov_CollectConfigRepoRefs_Good_AgentsBlock — the per-agent
// "agents:" map in agents.yaml contributes each agent's repos to the ref set,
// on top of the top-level "repos:" list.
func TestFetchLoopCov_CollectConfigRepoRefs_Good_AgentsBlock(t *testing.T) {
	raw := map[string]any{
		"repos": []any{"go-io", "core/go-store"},
		"agents": map[string]any{
			"virgil":    map[string]any{"repos": []any{"go-mlx"}},
			"hephestus": map[string]any{"repos": "core/go-rocm"},
			"broken":    "not-a-map",
		},
	}

	var refs []fetchRepoRef
	seen := map[string]bool{}
	fetchLoopCollectConfigRepoRefs(raw, func(org, repo string) {
		fetchLoopAppendRepoRef(seen, &refs, org, repo)
	})

	names := map[string]bool{}
	for _, ref := range refs {
		names[fetchLoopRepoName(ref)] = true
	}
	core.AssertTrue(t, names["core/go-io"])
	core.AssertTrue(t, names["core/go-store"])
	core.AssertTrue(t, names["core/go-mlx"])
	core.AssertTrue(t, names["core/go-rocm"])
}

// TestFetchLoopCov_CollectConfigRepoRefs_Bad_NoAgentsKey — when "agents" is
// absent the function returns after only the "repos:" list, leaving the
// agents loop untouched.
func TestFetchLoopCov_CollectConfigRepoRefs_Bad_NoAgentsKey(t *testing.T) {
	var refs []fetchRepoRef
	seen := map[string]bool{}
	fetchLoopCollectConfigRepoRefs(map[string]any{"repos": []any{"go-io"}}, func(org, repo string) {
		fetchLoopAppendRepoRef(seen, &refs, org, repo)
	})

	core.AssertLen(t, refs, 1)
	core.AssertEqual(t, "core/go-io", fetchLoopRepoName(refs[0]))
}

// TestFetchLoopCov_CollectRepoRefs_Good_AllShapes — fetchLoopCollectRepoRefs
// accepts a bare string, []string, []any and map[string]any, parsing each
// element through fetchLoopParseRepo.
func TestFetchLoopCov_CollectRepoRefs_Good_AllShapes(t *testing.T) {
	collect := func(value any) map[string]bool {
		var refs []fetchRepoRef
		seen := map[string]bool{}
		fetchLoopCollectRepoRefs(value, func(org, repo string) {
			fetchLoopAppendRepoRef(seen, &refs, org, repo)
		})
		names := map[string]bool{}
		for _, ref := range refs {
			names[fetchLoopRepoName(ref)] = true
		}
		return names
	}

	core.AssertTrue(t, collect("go-io")["core/go-io"])
	core.AssertTrue(t, collect([]string{"lthn/desktop"})["lthn/desktop"])
	core.AssertTrue(t, collect([]any{"go-mlx", 99})["core/go-mlx"])
	core.AssertTrue(t, collect(map[string]any{"go-store": 1})["core/go-store"])
}

// TestFetchLoopCov_CollectRepoRefs_Bad_UnsupportedType — an int value matches
// no switch arm, so nothing is added.
func TestFetchLoopCov_CollectRepoRefs_Bad_UnsupportedType(t *testing.T) {
	var refs []fetchRepoRef
	seen := map[string]bool{}
	fetchLoopCollectRepoRefs(42, func(org, repo string) {
		fetchLoopAppendRepoRef(seen, &refs, org, repo)
	})
	core.AssertLen(t, refs, 0)
}

// TestFetchLoopCov_ParseRepo_Good_OrgSlashRepo — a two-segment "org/repo"
// keeps the explicit org rather than defaulting to "core".
func TestFetchLoopCov_ParseRepo_Good_OrgSlashRepo(t *testing.T) {
	org, repo, ok := fetchLoopParseRepo("lthn/desktop")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "lthn", org)
	core.AssertEqual(t, "desktop", repo)
}

// TestFetchLoopCov_ParseRepo_Bad_Empty — a blank/whitespace value is rejected.
func TestFetchLoopCov_ParseRepo_Bad_Empty(t *testing.T) {
	_, _, ok := fetchLoopParseRepo("   ")
	core.AssertFalse(t, ok)
}

// TestFetchLoopCov_ParseRepo_Ugly_TooManySegments — three+ segments fall to
// the default arm and are rejected.
func TestFetchLoopCov_ParseRepo_Ugly_TooManySegments(t *testing.T) {
	_, _, ok := fetchLoopParseRepo("a/b/c")
	core.AssertFalse(t, ok)

	// A blank org segment in a two-part path is also invalid (validateName rejects "").
	_, _, badOrg := fetchLoopParseRepo("/desktop")
	core.AssertFalse(t, badOrg)
}

// TestFetchLoopCov_CollectWorkspaceRepoRefs_Good_ScansWorkspace — every
// org/repo directory two levels under the workspace root becomes a ref;
// files (non-dirs) at that depth are skipped.
func TestFetchLoopCov_CollectWorkspaceRepoRefs_Good_ScansWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := WorkspaceRoot()
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsRoot, "core", "go-io")).OK)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsRoot, "lthn", "desktop")).OK)
	// A file at the org/repo depth must be ignored.
	core.RequireTrue(t, fs.Write(core.JoinPath(wsRoot, "core", "stray.txt"), "x").OK)

	s := fetchLoopTestPrep(t.TempDir())
	var refs []fetchRepoRef
	seen := map[string]bool{}
	s.fetchLoopCollectWorkspaceRepoRefs(func(org, repo string) {
		fetchLoopAppendRepoRef(seen, &refs, org, repo)
	})

	names := map[string]bool{}
	for _, ref := range refs {
		names[fetchLoopRepoName(ref)] = true
	}
	core.AssertTrue(t, names["core/go-io"])
	core.AssertTrue(t, names["lthn/desktop"])
	core.AssertFalse(t, names["core/stray.txt"])
}

// TestFetchLoopCov_RepoRefs_Good_DedupesConfigAndWorkspace — fetchLoopRepoRefs
// merges configured + workspace refs and removes duplicates by org/repo key.
func TestFetchLoopCov_RepoRefs_Good_DedupesConfigAndWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := WorkspaceRoot()
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsRoot, "core", "go-io")).OK)

	codePath := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"repos:\n",
		"  - go-io\n",
		"  - lthn/desktop\n",
	)).OK)

	s := fetchLoopTestPrep(codePath)
	refs := s.fetchLoopRepoRefs()

	count := map[string]int{}
	for _, ref := range refs {
		count[fetchLoopRepoName(ref)]++
	}
	core.AssertEqual(t, 1, count["core/go-io"]) // configured + workspace → one entry
	core.AssertEqual(t, 1, count["lthn/desktop"])
}

// TestFetchLoopCov_Interval_Good_ConfigYAMLDispatch — with no store override
// the interval is read from the dispatch.fetch_interval key in agents.yaml.
func TestFetchLoopCov_Interval_Good_ConfigYAMLDispatch(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 90s\n",
	)).OK)

	s := fetchLoopTestPrep(t.TempDir())
	core.AssertEqual(t, 90*time.Second, s.fetchLoopInterval())
}

// TestFetchLoopCov_Interval_Bad_FallsBackToDefault — an agents.yaml with no
// fetch_interval anywhere yields the package default.
func TestFetchLoopCov_Interval_Bad_FallsBackToDefault(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), "version: 1\nrepos:\n  - go-io\n").OK)

	s := fetchLoopTestPrep(t.TempDir())
	core.AssertEqual(t, fetchLoopDefaultInterval, s.fetchLoopInterval())
}

// TestFetchLoopCov_ReadConfig_Bad_MissingFile — a non-existent path returns an
// empty map rather than erroring.
func TestFetchLoopCov_ReadConfig_Bad_MissingFile(t *testing.T) {
	raw := fetchLoopReadConfig(core.JoinPath(t.TempDir(), "absent.yaml"))
	core.AssertLen(t, raw, 0)
}

// TestFetchLoopCov_ReadConfig_Ugly_InvalidYAML — malformed YAML also yields an
// empty map (the unmarshal error is swallowed).
func TestFetchLoopCov_ReadConfig_Ugly_InvalidYAML(t *testing.T) {
	path := core.JoinPath(t.TempDir(), "bad.yaml")
	core.RequireTrue(t, fs.Write(path, "version: 1\n  - broken: [\n").OK)
	raw := fetchLoopReadConfig(path)
	core.AssertLen(t, raw, 0)
}

// TestFetchLoopCov_ConfigPaths_Good_DedupesAndTrims — fetchLoopConfigPaths
// returns the workspace agents path plus the codePath-derived path, with no
// duplicates and no blank entries.
func TestFetchLoopCov_ConfigPaths_Good_DedupesAndTrims(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	s := fetchLoopTestPrep(codePath)
	paths := s.fetchLoopConfigPaths()

	core.AssertNotEmpty(t, paths)
	seen := map[string]bool{}
	for _, p := range paths {
		core.AssertNotEqual(t, "", p)
		core.AssertFalse(t, seen[p])
		seen[p] = true
	}
	core.AssertTrue(t, seen[core.JoinPath(codePath, "core", "agent", ".core", "agents.yaml")])
}
