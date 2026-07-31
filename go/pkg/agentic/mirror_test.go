// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// initBareRepo creates a minimal git repo with one commit and returns its path.
func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitEnv := []string{
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Test")
	run("git", "config", "user.email", "test@test.com")

	// Create a file and commit
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "README.md"), "# Test").OK)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")
	return dir
}

// --- hasRemote ---

func TestMirror_HasRemote_Good_OriginExists(t *testing.T) {
	dir := initBareRepo(t)
	// origin won't exist for a fresh repo, so add it
	testCore.Process().RunIn(context.Background(), dir, "git", "remote", "add", "origin", "https://example.com/repo.git")

	core.AssertTrue(t, testPrep.hasRemote(dir, "origin"))
}

func TestMirror_HasRemote_Good_CustomRemote(t *testing.T) {
	dir := initBareRepo(t)
	testCore.Process().RunIn(context.Background(), dir, "git", "remote", "add", "github", "https://github.com/test/repo.git")

	core.AssertTrue(t, testPrep.hasRemote(dir, "github"))
}

func TestMirror_HasRemote_Bad_NoSuchRemote(t *testing.T) {
	dir := initBareRepo(t)
	core.AssertFalse(
		t,
		testPrep.hasRemote(dir, "nonexistent"),
	)
}

func TestMirror_HasRemote_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir() // plain directory, no .git
	core.AssertFalse(
		t,
		testPrep.hasRemote(dir, "origin"),
	)
}

func TestMirror_HasRemote_Ugly_EmptyDir(t *testing.T) {
	// Empty dir defaults to cwd which may or may not be a repo.
	// Just ensure no panic.
	core.AssertNotPanics(t, func() {
		testPrep.hasRemote("", "origin")
	})
}

// --- commitsAhead ---

func TestMirror_CommitsAhead_Good_OneAhead(t *testing.T) {
	dir := initBareRepo(t)

	// Create a branch at the current commit to act as "base"
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}

	run("git", "branch", "base")

	// Add a commit on main
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "new.txt"), "data").OK)
	run("git", "add", "new.txt")
	run("git", "commit", "-m", "second commit")

	ahead := testPrep.commitsAhead(dir, "base", "main")
	core.AssertEqual(t, 1, ahead)
}

func TestMirror_CommitsAhead_Good_ThreeAhead(t *testing.T) {
	dir := initBareRepo(t)
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}

	run("git", "branch", "base")

	for i := range 3 {
		name := core.JoinPath(dir, "file"+string(rune('a'+i))+".txt")
		core.RequireTrue(t, fs.Write(name, "content").OK)
		run("git", "add", ".")
		run("git", "commit", "-m", "commit "+string(rune('0'+i)))
	}

	ahead := testPrep.commitsAhead(dir, "base", "main")
	core.AssertEqual(t, 3, ahead)
}

func TestMirror_CommitsAhead_Good_ZeroAhead(t *testing.T) {
	dir := initBareRepo(t)
	// Same ref on both sides
	ahead := testPrep.commitsAhead(dir, "main", "main")
	core.AssertEqual(t, 0, ahead)
}

func TestMirror_CommitsAhead_Bad_InvalidRef(t *testing.T) {
	dir := initBareRepo(t)
	ahead := testPrep.commitsAhead(dir, "nonexistent-ref", "main")
	core.AssertEqual(t, 0, ahead)
}

func TestMirror_CommitsAhead_Bad_NotARepo(t *testing.T) {
	ahead := testPrep.commitsAhead(t.TempDir(), "main", "dev")
	core.AssertEqual(
		t,
		0,
		ahead,
	)
}

func TestMirror_CommitsAhead_Ugly_EmptyDir(t *testing.T) {
	ahead := testPrep.commitsAhead("", "a", "b")
	core.AssertEqual(
		t,
		0,
		ahead,
	)
}

// --- filesChanged ---

func TestMirror_FilesChanged_Good_OneFile(t *testing.T) {
	dir := initBareRepo(t)
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}

	run("git", "branch", "base")

	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "changed.txt"), "new").OK)
	run("git", "add", "changed.txt")
	run("git", "commit", "-m", "add file")

	files := testPrep.filesChanged(dir, "base", "main")
	core.AssertEqual(t, 1, files)
}

func TestMirror_FilesChanged_Good_MultipleFiles(t *testing.T) {
	dir := initBareRepo(t)
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}

	run("git", "branch", "base")

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		core.RequireTrue(t, fs.Write(core.JoinPath(dir, name), "package main").OK)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "add three files")

	files := testPrep.filesChanged(dir, "base", "main")
	core.AssertEqual(t, 3, files)
}

func TestMirror_FilesChanged_Good_NoChanges(t *testing.T) {
	dir := initBareRepo(t)
	files := testPrep.filesChanged(dir, "main", "main")
	core.AssertEqual(t, 0, files)
}

func TestMirror_FilesChanged_Bad_InvalidRef(t *testing.T) {
	dir := initBareRepo(t)
	files := testPrep.filesChanged(dir, "nonexistent", "main")
	core.AssertEqual(t, 0, files)
}

func TestMirror_FilesChanged_Bad_NotARepo(t *testing.T) {
	files := testPrep.filesChanged(t.TempDir(), "main", "dev")
	core.AssertEqual(
		t,
		0,
		files,
	)
}

func TestMirror_FilesChanged_Ugly_EmptyDir(t *testing.T) {
	files := testPrep.filesChanged("", "a", "b")
	core.AssertEqual(
		t,
		0,
		files,
	)
}

// --- extractJSONField (extending existing 91% coverage) ---

func TestMirror_ExtractJSONField_Good_ArrayFirstItem(t *testing.T) {
	json := `[{"url":"https://github.com/test/pr/1","title":"Fix bug"}]`
	core.AssertEqual(
		t,
		"https://github.com/test/pr/1",
		extractJSONField(json, "url"),
	)
}

func TestMirror_ExtractJSONField_Good_ObjectField(t *testing.T) {
	json := `{"name":"test-repo","status":"active"}`
	core.AssertEqual(
		t,
		"test-repo",
		extractJSONField(json, "name"),
	)
}

func TestMirror_ExtractJSONField_Good_ArrayMultipleItems(t *testing.T) {
	json := `[{"id":"first"},{"id":"second"}]`
	// Should return the first match
	core.AssertEqual(
		t,
		"first",
		extractJSONField(json, "id"),
	)
}

func TestMirror_ExtractJSONField_Bad_EmptyJSON(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		extractJSONField("", "url"),
	)
}

func TestMirror_ExtractJSONField_Bad_EmptyField(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		extractJSONField(`{"url":"test"}`, ""),
	)
}

func TestMirror_ExtractJSONField_Bad_FieldNotFound(t *testing.T) {
	json := `{"name":"test"}`
	core.AssertEqual(
		t,
		"",
		extractJSONField(json, "missing"),
	)
}

func TestMirror_ExtractJSONField_Bad_InvalidJSON(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		extractJSONField("not json at all", "url"),
	)
}

func TestMirror_ExtractJSONField_Ugly_EmptyArray(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		extractJSONField("[]", "url"),
	)
}

func TestMirror_ExtractJSONField_Ugly_EmptyObject(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		extractJSONField("{}", "url"),
	)
}

func TestMirror_ExtractJSONField_Ugly_NumericValue(t *testing.T) {
	// Field exists but is not a string — should return ""
	json := `{"count":42}`
	core.AssertEqual(
		t,
		"",
		extractJSONField(json, "count"),
	)
}

func TestMirror_ExtractJSONField_Ugly_NullValue(t *testing.T) {
	json := `{"url":null}`
	core.AssertEqual(
		t,
		"",
		extractJSONField(json, "url"),
	)
}

// --- DefaultBranch ---

func TestPaths_DefaultBranch_Good_MainBranch(t *testing.T) {
	dir := initBareRepo(t)
	// initBareRepo creates with -b main
	branch := testPrep.DefaultBranch(dir)
	core.AssertEqual(t, "main", branch)
}

func TestPaths_DefaultBranch_Bad_NotARepo(t *testing.T) {
	dir := t.TempDir()
	// Falls back to "main" when detection fails
	branch := testPrep.DefaultBranch(dir)
	core.AssertEqual(t, "main", branch)
}

// --- listLocalRepos ---

func TestMirror_ListLocalRepos_Good_FindsRepos(t *testing.T) {
	base := t.TempDir()

	// Create two git repos under base
	for _, name := range []string{"repo-a", "repo-b"} {
		repoDir := core.JoinPath(base, name)
		testCore.Process().Run(context.Background(), "git", "init", repoDir)
	}

	// Create a non-repo directory
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(base, "not-a-repo")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	repos := s.listLocalRepos(base)
	core.AssertContains(t, repos, "repo-a")
	core.AssertContains(t, repos, "repo-b")
	core.AssertNotContains(t, repos, "not-a-repo")
}

func TestMirror_ListLocalRepos_Bad_EmptyDir(t *testing.T) {
	base := t.TempDir()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	repos := s.listLocalRepos(base)
	core.AssertEmpty(t, repos)
}

func TestMirror_ListLocalRepos_Bad_NonExistentDir(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	repos := s.listLocalRepos("/nonexistent/path/that/doesnt/exist")
	core.AssertNil(t, repos)
}

// --- GitHubOrg ---

func TestPaths_GitHubOrg_Good_Default(t *testing.T) {
	t.Setenv("GITHUB_ORG", "")
	core.AssertEqual(
		t,
		"dAppCore",
		GitHubOrg(),
	)
}

func TestPaths_GitHubOrg_Good_Custom(t *testing.T) {
	t.Setenv("GITHUB_ORG", "my-org")
	core.AssertEqual(
		t,
		"my-org",
		GitHubOrg(),
	)
}

// --- listLocalRepos Ugly ---

func TestMirror_ListLocalRepos_Ugly_Case(t *testing.T) {
	base := t.TempDir()

	// Create two git repos
	for _, name := range []string{"real-repo-a", "real-repo-b"} {
		repoDir := core.JoinPath(base, name)
		testCore.Process().Run(context.Background(), "git", "init", repoDir)
	}

	// Create non-git directories (no .git inside)
	for _, name := range []string{"plain-dir", "another-dir"} {
		core.RequireTrue(t, fs.EnsureDir(core.JoinPath(base, name)).OK)
	}

	// Create a regular file (not a directory)
	core.RequireTrue(t, fs.Write(core.JoinPath(base, "some-file.txt"), "hello").OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	repos := s.listLocalRepos(base)
	core.AssertContains(t, repos, "real-repo-a")
	core.AssertContains(t, repos, "real-repo-b")
	core.AssertNotContains(t, repos, "plain-dir")
	core.AssertNotContains(t, repos, "another-dir")
	core.AssertNotContains(t, repos, "some-file.txt")
	core.AssertLen(t, repos, 2)
}
