// SPDX-License-Identifier: EUPL-1.2

// Orchestration + GitHub-PR coverage for the mirror flow. The helper-level
// funcs (hasRemote / commitsAhead / filesChanged / listLocalRepos) are
// covered in mirror_test.go; this file drives the top-level mirror loop and
// the createGitHubPR / ensureDevBranch side-effecting ops.
//
// Seams: a real git repo with a LOCAL bare "github" remote (so fetch / push /
// rev-list run against a real remote, no network) + a fake `gh` on PATH (so
// createGitHubPR's gh-list / gh-create branches run deterministically without
// touching the real GitHub API or launching anything). process.RunIn honours
// a t.Setenv("PATH", ...) override, so no production seam is needed here.

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

var mirrorGitEnv = []string{
	"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
	"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
}

func mirrorGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	r := testCore.Process().RunWithEnv(context.Background(), dir, mirrorGitEnv, args[0], args[1:]...)
	if !r.OK {
		t.Fatalf("git %v failed: %v", args, r.Value)
	}
}

// initRepoWithBareGithub creates repoDir with one commit on main, a local bare
// repo as the "github" remote already holding that commit, then adds extra
// commits on local main so commitsAhead(github/main, main) > 0. Returns the
// number of commits ahead and the count of distinct files changed.
func initRepoWithBareGithub(t *testing.T, repoDir string, extraCommits int) {
	t.Helper()
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	mirrorGit(t, repoDir, "git", "init", "-b", "main")
	mirrorGit(t, repoDir, "git", "config", "user.name", "Test")
	mirrorGit(t, repoDir, "git", "config", "user.email", "test@test.com")
	core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# Test").OK)
	mirrorGit(t, repoDir, "git", "add", "README.md")
	mirrorGit(t, repoDir, "git", "commit", "-m", "initial commit")

	// Bare remote seeded from the initial commit.
	bare := core.JoinPath(t.TempDir(), "github.git")
	mirrorGit(t, repoDir, "git", "init", "--bare", bare)
	mirrorGit(t, repoDir, "git", "remote", "add", "github", bare)
	mirrorGit(t, repoDir, "git", "push", "github", "main")

	// Diverge local main ahead of github/main.
	for i := range extraCommits {
		name := core.Concat("file", string(rune('a'+i)), ".txt")
		core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, name), "data").OK)
		mirrorGit(t, repoDir, "git", "add", ".")
		mirrorGit(t, repoDir, "git", "commit", "-m", core.Concat("commit ", name))
	}
	// Refresh remote-tracking refs so github/main resolves locally.
	mirrorGit(t, repoDir, "git", "fetch", "github")
}

// writeFakeGh drops a fake `gh` binary on PATH whose behaviour is controlled by
// the GH_FIXTURE_MODE env var read at call time:
//
//	list-has-url  → `gh pr list` prints a JSON array with a url, create unused
//	create-ok     → `gh pr list` prints [], `gh pr create` prints a PR url
//	create-fail   → `gh pr list` prints [], `gh pr create` exits 1
func writeFakeGh(t *testing.T, mode string) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/bin/sh
mode="$GH_FIXTURE_MODE"
case "$1 $2" in
  "pr list")
    if [ "$mode" = "list-has-url" ]; then
      echo '[{"url":"https://github.com/dAppCore/go-io/pull/7"}]'
    else
      echo '[]'
    fi
    ;;
  "pr create")
    if [ "$mode" = "create-fail" ]; then
      echo "gh: could not create pull request" >&2
      exit 1
    fi
    echo "https://github.com/dAppCore/go-io/pull/9"
    ;;
  *)
    echo "unexpected gh args: $*" >&2
    exit 2
    ;;
esac
`
	core.RequireTrue(t, core.WriteFile(core.JoinPath(bin, "gh"), []byte(script), 0o755).OK)
	t.Setenv("GH_FIXTURE_MODE", mode)
	t.Setenv("PATH", bin+":"+core.Env("PATH"))
}

// --- createGitHubPR ---

// TestMirror_CreateGitHubPR_Good_ExistingPRReused — when `gh pr list` already
// returns an open PR url, createGitHubPR returns it without creating one.
func TestMirror_CreateGitHubPR_Good_ExistingPRReused(t *testing.T) {
	writeFakeGh(t, "list-has-url")
	repoDir := t.TempDir()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}

	r := s.createGitHubPR(context.Background(), repoDir, "go-io", 3, 12)
	core.RequireTrue(t, r.OK)
	url, ok := r.Value.(string)
	core.RequireTrue(t, ok)
	core.AssertContains(t, url, "/pull/7")
}

// TestMirror_CreateGitHubPR_Good_CreatesNew — no existing PR → gh create runs
// and its last-line url is returned.
func TestMirror_CreateGitHubPR_Good_CreatesNew(t *testing.T) {
	writeFakeGh(t, "create-ok")
	repoDir := t.TempDir()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}

	r := s.createGitHubPR(context.Background(), repoDir, "go-io", 1, 2)
	core.RequireTrue(t, r.OK)
	url, ok := r.Value.(string)
	core.RequireTrue(t, ok)
	core.AssertContains(t, url, "/pull/9")
}

// TestMirror_CreateGitHubPR_Bad_CreateFails — gh create exits non-zero →
// createGitHubPR surfaces a typed Fail.
func TestMirror_CreateGitHubPR_Bad_CreateFails(t *testing.T) {
	writeFakeGh(t, "create-fail")
	repoDir := t.TempDir()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}

	r := s.createGitHubPR(context.Background(), repoDir, "go-io", 1, 2)
	core.AssertFalse(t, r.OK)
}

// --- ensureDevBranch ---

// TestMirror_EnsureDevBranch_Good_PushesHead — ensureDevBranch pushes HEAD to
// the github remote's dev ref; against a local bare remote the push succeeds
// and the ref is created.
func TestMirror_EnsureDevBranch_Good_PushesHead(t *testing.T) {
	repoDir := t.TempDir()
	initRepoWithBareGithub(t, repoDir, 1)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	core.AssertNotPanics(t, func() { s.ensureDevBranch(repoDir) })

	// The dev ref now exists on the github remote.
	r := testCore.Process().RunIn(context.Background(), repoDir, "git", "ls-remote", "--heads", "github", "dev")
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, r.Value.(string), "refs/heads/dev")
}

// --- mirror (orchestration) ---

// mirrorSubsystem builds a PrepSubsystem whose codePath is a temp root; the
// mirror loop scans <codePath>/core/<repo>, so callers must create fixtures
// there. Returns the subsystem and the core/ base path.
func mirrorSubsystem(t *testing.T) (*PrepSubsystem, string) {
	t.Helper()
	codePath := t.TempDir()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: codePath}
	return s, core.JoinPath(codePath, "core")
}

// TestMirror_Mirror_Good_DryRunReportsAhead — a repo with a github remote and
// commits ahead, mirrored in DryRun, reports the ahead/files counts and a "dry
// run" skip without pushing. Asserts on Synced[0] so a misplaced fixture (which
// would hit the no-remote skip and still return OK) fails loudly.
func TestMirror_Mirror_Good_DryRunReportsAhead(t *testing.T) {
	s, base := mirrorSubsystem(t)
	repoDir := core.JoinPath(base, "go-io")
	initRepoWithBareGithub(t, repoDir, 2)

	r := s.mirror(context.Background(), MirrorInput{Repo: "go-io", DryRun: true})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(MirrorOutput)
	core.RequireTrue(t, ok)
	core.RequireTrue(t, len(out.Synced) == 1)
	core.AssertEqual(t, "go-io", out.Synced[0].Repo)
	core.AssertEqual(t, 2, out.Synced[0].CommitsAhead)
	core.AssertEqual(t, "dry run", out.Synced[0].Skipped)
	core.AssertFalse(t, out.Synced[0].Pushed)
}

// TestMirror_Mirror_Good_ExceedsFileLimit — when the changed-file count exceeds
// MaxFiles the repo is reported with the limit-exceeded reason and not pushed,
// even outside DryRun (the limit check precedes the push).
func TestMirror_Mirror_Good_ExceedsFileLimit(t *testing.T) {
	s, base := mirrorSubsystem(t)
	repoDir := core.JoinPath(base, "go-io")
	initRepoWithBareGithub(t, repoDir, 3) // 3 distinct files ahead

	r := s.mirror(context.Background(), MirrorInput{Repo: "go-io", MaxFiles: 1})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(MirrorOutput)
	core.RequireTrue(t, ok)
	core.RequireTrue(t, len(out.Synced) == 1)
	core.AssertContains(t, out.Synced[0].Skipped, "exceeds limit")
	core.AssertFalse(t, out.Synced[0].Pushed)
}

// TestMirror_Mirror_Skip_NoGithubRemote — a repo without a github remote is
// recorded in Skipped, not Synced.
func TestMirror_Mirror_Skip_NoGithubRemote(t *testing.T) {
	s, base := mirrorSubsystem(t)
	repoDir := core.JoinPath(base, "go-io")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	mirrorGit(t, repoDir, "git", "init", "-b", "main")

	r := s.mirror(context.Background(), MirrorInput{Repo: "go-io"})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(MirrorOutput)
	core.RequireTrue(t, ok)
	core.AssertEmpty(t, out.Synced)
	core.RequireTrue(t, len(out.Skipped) == 1)
	core.AssertContains(t, out.Skipped[0], "no github remote")
}

// TestMirror_Mirror_Good_PushAndPR — full non-dry-run leg: push to the local
// bare remote succeeds and the fake gh creates a PR. Synced[0] is pushed with
// the PR url.
func TestMirror_Mirror_Good_PushAndPR(t *testing.T) {
	writeFakeGh(t, "create-ok")
	s, base := mirrorSubsystem(t)
	repoDir := core.JoinPath(base, "go-io")
	initRepoWithBareGithub(t, repoDir, 1)

	r := s.mirror(context.Background(), MirrorInput{Repo: "go-io"})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(MirrorOutput)
	core.RequireTrue(t, ok)
	core.RequireTrue(t, len(out.Synced) == 1)
	core.AssertTrue(t, out.Synced[0].Pushed)
	core.AssertContains(t, out.Synced[0].PRURL, "/pull/9")
}
