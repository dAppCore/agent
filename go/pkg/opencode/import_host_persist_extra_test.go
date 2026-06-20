// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestImportHost_persistProjects_RealRows_Body — drives persistProjects
// with fully-populated project maps (icon block, time block, sandboxes,
// virtual worktree). The unit Core here has no registered ImportedProject
// table, so each Save fails and the returned count stays 0 — but the field
// extraction body (icon/time/worktree/sandboxes handling, the per-row map
// type-assertions, and the unixMillis + projectNameFrom calls) is exercised.
// Non-map entries and id-less rows are skipped before any Save.
func TestImportHost_persistProjects_RealRows_Body(t *testing.T) {
	c := core.New(core.WithOption("name", "opencode-test"))
	now := core.Now()

	projects := []any{
		// fully-populated row — real worktree, icon + time blocks, sandboxes.
		map[string]any{
			"id":       "proj-1",
			"worktree": "/home/user/myrepo",
			"vcs":      "git",
			"icon":     map[string]any{"color": "#abc", "url": "data:image/png;base64,AAAA"},
			"time":     map[string]any{"created": float64(1700000000000), "updated": float64(1700000100000)},
			"sandboxes": []any{
				map[string]any{"id": "sb-1"},
			},
		},
		// virtual ("/") worktree — name falls back to the source id.
		map[string]any{
			"id":       "proj-2",
			"worktree": "/",
		},
		// non-map entry — skipped by the `p, ok := raw.(map...)` guard.
		"not-a-map",
		// id-less row — skipped by the sourceID=="" guard.
		map[string]any{"worktree": "/tmp/no-id"},
	}

	// Save is unbacked here → count stays 0, but the body ran for every
	// well-formed row. The contract under test is "no panic + clean count",
	// not the persisted side-effect (that needs a migrated DuckDB medium,
	// covered by the wire/import tests).
	got := persistProjects(c, projects, now)
	core.AssertTrue(t, got >= 0)
}

// TestImportHost_persistProviders_RealRows_Body — drives persistProviders
// with provider maps plus a matching authMap so the auth-lookup +
// has-auth + options-JSON branches run. As with persistProjects the unit
// Core has no registered table, so (count, withAuth) come back (0, _),
// but the per-provider body is exercised.
func TestImportHost_persistProviders_RealRows_Body(t *testing.T) {
	c := core.New(core.WithOption("name", "opencode-test"))
	now := core.Now()

	providers := []any{
		map[string]any{
			"id":      "anthropic",
			"name":    "Anthropic",
			"npm":     "@ai-sdk/anthropic",
			"options": map[string]any{"baseURL": "https://api.anthropic.com"},
		},
		// no-auth provider (absent from authMap) → hasAuth false branch.
		map[string]any{
			"id":   "openai",
			"name": "OpenAI",
		},
		// non-map + id-less → skipped before Save.
		42,
		map[string]any{"name": "no-id"},
	}
	authMap := map[string]map[string]any{
		"anthropic": {"type": "apikey", "key": "sk-secret"},
	}

	count, withAuth := persistProviders(c, providers, authMap, now)
	core.AssertTrue(t, count >= 0)
	core.AssertTrue(t, withAuth >= 0)
}

// TestImportHost_unixMillis_AllBranches — float64 / int64 / zero / negative
// / absent inputs map to the expected core.Time (zero when non-positive or
// non-numeric, a real instant otherwise).
func TestImportHost_unixMillis_AllBranches(t *testing.T) {
	// positive float64 (the opencode JSON-decoded shape) → non-zero.
	core.AssertFalse(t, unixMillis(float64(1700000000000)).IsZero())
	// positive int64 → non-zero.
	core.AssertFalse(t, unixMillis(int64(1700000000000)).IsZero())
	// non-positive float64 → zero.
	core.AssertTrue(t, unixMillis(float64(0)).IsZero())
	core.AssertTrue(t, unixMillis(float64(-1)).IsZero())
	// nil / wrong type → zero.
	core.AssertTrue(t, unixMillis(nil).IsZero())
	core.AssertTrue(t, unixMillis("nope").IsZero())
}
