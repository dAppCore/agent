// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"testing"

	core "dappco.re/go"
)

// TestHarvest_DefaultBranch_Good_Case — the default branch resolves via
// origin/HEAD on a clone, and via the main/master probe on a remote-less repo.
func TestHarvest_DefaultBranch_Good_Case(t *testing.T) {
	sourceDir, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	// The clone carries origin/HEAD → resolved through the origin/ prefix path.
	core.AssertEqual(t, "main", testMon.defaultBranch(repoDir))

	// The bare source has no remote → resolved through the main/master probe.
	core.AssertEqual(t, "main", testMon.defaultBranch(sourceDir))
}

// TestHarvest_DefaultBranch_Bad_NoRepo — a directory that is not a git repo
// falls back to "main".
func TestHarvest_DefaultBranch_Bad_NoRepo(t *testing.T) {
	core.AssertEqual(t, "main", testMon.defaultBranch(t.TempDir()))
}
