// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initBareRepo creates a minimal git repo with one commit and returns its path.
func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Test")
	run("git", "config", "user.email", "test@test.com")

	// Create a file and commit
	require.True(t, fs.Write(filepath.Join(dir, "README.md"), "# Test").OK)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")
	return dir
}

// --- hasRemote ---

func TestMirror_HasRemote_Good_OriginExists(t *testing.T) {
	dir := initBareRepo(t)
	// origin won't exist for a fresh repo, so add it
	cmd := exec.Command("git", "remote", "add", "origin", "https://example.com/repo.git")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	assert.True(t, hasRemote(dir, "origin"))
}

func TestMirror_HasRemote_Good_CustomRemote(t *testing.T) {
	dir := initBareRepo(t)
	cmd := exec.Command("git", "remote", "add", "github", "https://github.com/test/repo.git")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	assert.True(t, hasRemote(dir, "github"))
}

func TestMirror_HasRemote_Bad_NoSuchRemote(t *testing.T) {
	dir := initBareRepo(t)
	assert.False(t, hasRemote(dir, "nonexistent"))
}

func TestMirror_HasRemote_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir() // plain directory, no .git
	assert.False(t, hasRemote(dir, "origin"))
}

func TestMirror_HasRemote_Ugly_EmptyDir(t *testing.T) {
	// Empty dir defaults to cwd which may or may not be a repo.
	// Just ensure no panic.
	assert.NotPanics(t, func() {
		hasRemote("", "origin")
	})
}

// --- commitsAhead ---

func TestMirror_CommitsAhead_Good_OneAhead(t *testing.T) {
	dir := initBareRepo(t)

	// Create a branch at the current commit to act as "base"
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}

	run("git", "branch", "base")

	// Add a commit on main
	require.True(t, fs.Write(filepath.Join(dir, "new.txt"), "data").OK)
	run("git", "add", "new.txt")
	run("git", "commit", "-m", "second commit")

	ahead := commitsAhead(dir, "base", "main")
	assert.Equal(t, 1, ahead)
}

func TestMirror_CommitsAhead_Good_ThreeAhead(t *testing.T) {
	dir := initBareRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}

	run("git", "branch", "base")

	for i := 0; i < 3; i++ {
		name := filepath.Join(dir, "file"+string(rune('a'+i))+".txt")
		require.True(t, fs.Write(name, "content").OK)
		run("git", "add", ".")
		run("git", "commit", "-m", "commit "+string(rune('0'+i)))
	}

	ahead := commitsAhead(dir, "base", "main")
	assert.Equal(t, 3, ahead)
}

func TestMirror_CommitsAhead_Good_ZeroAhead(t *testing.T) {
	dir := initBareRepo(t)
	// Same ref on both sides
	ahead := commitsAhead(dir, "main", "main")
	assert.Equal(t, 0, ahead)
}

func TestMirror_CommitsAhead_Bad_InvalidRef(t *testing.T) {
	dir := initBareRepo(t)
	ahead := commitsAhead(dir, "nonexistent-ref", "main")
	assert.Equal(t, 0, ahead)
}

func TestMirror_CommitsAhead_Bad_NotARepo(t *testing.T) {
	ahead := commitsAhead(t.TempDir(), "main", "dev")
	assert.Equal(t, 0, ahead)
}

func TestMirror_CommitsAhead_Ugly_EmptyDir(t *testing.T) {
	ahead := commitsAhead("", "a", "b")
	assert.Equal(t, 0, ahead)
}

// --- filesChanged ---

func TestMirror_FilesChanged_Good_OneFile(t *testing.T) {
	dir := initBareRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}

	run("git", "branch", "base")

	require.True(t, fs.Write(filepath.Join(dir, "changed.txt"), "new").OK)
	run("git", "add", "changed.txt")
	run("git", "commit", "-m", "add file")

	files := filesChanged(dir, "base", "main")
	assert.Equal(t, 1, files)
}

func TestMirror_FilesChanged_Good_MultipleFiles(t *testing.T) {
	dir := initBareRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}

	run("git", "branch", "base")

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		require.True(t, fs.Write(filepath.Join(dir, name), "package main").OK)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "add three files")

	files := filesChanged(dir, "base", "main")
	assert.Equal(t, 3, files)
}

func TestMirror_FilesChanged_Good_NoChanges(t *testing.T) {
	dir := initBareRepo(t)
	files := filesChanged(dir, "main", "main")
	assert.Equal(t, 0, files)
}

func TestMirror_FilesChanged_Bad_InvalidRef(t *testing.T) {
	dir := initBareRepo(t)
	files := filesChanged(dir, "nonexistent", "main")
	assert.Equal(t, 0, files)
}

func TestMirror_FilesChanged_Bad_NotARepo(t *testing.T) {
	files := filesChanged(t.TempDir(), "main", "dev")
	assert.Equal(t, 0, files)
}

func TestMirror_FilesChanged_Ugly_EmptyDir(t *testing.T) {
	files := filesChanged("", "a", "b")
	assert.Equal(t, 0, files)
}

// --- extractJSONField (extending existing 91% coverage) ---

func TestMirror_ExtractJSONField_Good_ArrayFirstItem(t *testing.T) {
	json := `[{"url":"https://github.com/test/pr/1","title":"Fix bug"}]`
	assert.Equal(t, "https://github.com/test/pr/1", extractJSONField(json, "url"))
}

func TestMirror_ExtractJSONField_Good_ObjectField(t *testing.T) {
	json := `{"name":"test-repo","status":"active"}`
	assert.Equal(t, "test-repo", extractJSONField(json, "name"))
}

func TestMirror_ExtractJSONField_Good_ArrayMultipleItems(t *testing.T) {
	json := `[{"id":"first"},{"id":"second"}]`
	// Should return the first match
	assert.Equal(t, "first", extractJSONField(json, "id"))
}

func TestMirror_ExtractJSONField_Bad_EmptyJSON(t *testing.T) {
	assert.Equal(t, "", extractJSONField("", "url"))
}

func TestMirror_ExtractJSONField_Bad_EmptyField(t *testing.T) {
	assert.Equal(t, "", extractJSONField(`{"url":"test"}`, ""))
}

func TestMirror_ExtractJSONField_Bad_FieldNotFound(t *testing.T) {
	json := `{"name":"test"}`
	assert.Equal(t, "", extractJSONField(json, "missing"))
}

func TestMirror_ExtractJSONField_Bad_InvalidJSON(t *testing.T) {
	assert.Equal(t, "", extractJSONField("not json at all", "url"))
}

func TestMirror_ExtractJSONField_Ugly_EmptyArray(t *testing.T) {
	assert.Equal(t, "", extractJSONField("[]", "url"))
}

func TestMirror_ExtractJSONField_Ugly_EmptyObject(t *testing.T) {
	assert.Equal(t, "", extractJSONField("{}", "url"))
}

func TestMirror_ExtractJSONField_Ugly_NumericValue(t *testing.T) {
	// Field exists but is not a string — should return ""
	json := `{"count":42}`
	assert.Equal(t, "", extractJSONField(json, "count"))
}

func TestMirror_ExtractJSONField_Ugly_NullValue(t *testing.T) {
	json := `{"url":null}`
	assert.Equal(t, "", extractJSONField(json, "url"))
}

// --- DefaultBranch ---

func TestPaths_DefaultBranch_Good_MainBranch(t *testing.T) {
	dir := initBareRepo(t)
	// initBareRepo creates with -b main
	branch := DefaultBranch(dir)
	assert.Equal(t, "main", branch)
}

func TestPaths_DefaultBranch_Bad_NotARepo(t *testing.T) {
	dir := t.TempDir()
	// Falls back to "main" when detection fails
	branch := DefaultBranch(dir)
	assert.Equal(t, "main", branch)
}

// --- listLocalRepos ---

func TestMirror_ListLocalRepos_Good_FindsRepos(t *testing.T) {
	base := t.TempDir()

	// Create two git repos under base
	for _, name := range []string{"repo-a", "repo-b"} {
		repoDir := filepath.Join(base, name)
		cmd := exec.Command("git", "init", repoDir)
		require.NoError(t, cmd.Run())
	}

	// Create a non-repo directory
	require.True(t, fs.EnsureDir(filepath.Join(base, "not-a-repo")).OK)

	s := &PrepSubsystem{}
	repos := s.listLocalRepos(base)
	assert.Contains(t, repos, "repo-a")
	assert.Contains(t, repos, "repo-b")
	assert.NotContains(t, repos, "not-a-repo")
}

func TestMirror_ListLocalRepos_Bad_EmptyDir(t *testing.T) {
	base := t.TempDir()
	s := &PrepSubsystem{}
	repos := s.listLocalRepos(base)
	assert.Empty(t, repos)
}

func TestMirror_ListLocalRepos_Bad_NonExistentDir(t *testing.T) {
	s := &PrepSubsystem{}
	repos := s.listLocalRepos("/nonexistent/path/that/doesnt/exist")
	assert.Nil(t, repos)
}

// --- GitHubOrg ---

func TestPaths_GitHubOrg_Good_Default(t *testing.T) {
	t.Setenv("GITHUB_ORG", "")
	assert.Equal(t, "dAppCore", GitHubOrg())
}

func TestPaths_GitHubOrg_Good_Custom(t *testing.T) {
	t.Setenv("GITHUB_ORG", "my-org")
	assert.Equal(t, "my-org", GitHubOrg())
}

// --- listLocalRepos Ugly ---

func TestMirror_ListLocalRepos_Ugly(t *testing.T) {
	base := t.TempDir()

	// Create two git repos
	for _, name := range []string{"real-repo-a", "real-repo-b"} {
		repoDir := filepath.Join(base, name)
		cmd := exec.Command("git", "init", repoDir)
		require.NoError(t, cmd.Run())
	}

	// Create non-git directories (no .git inside)
	for _, name := range []string{"plain-dir", "another-dir"} {
		require.True(t, fs.EnsureDir(filepath.Join(base, name)).OK)
	}

	// Create a regular file (not a directory)
	require.True(t, fs.Write(filepath.Join(base, "some-file.txt"), "hello").OK)

	s := &PrepSubsystem{}
	repos := s.listLocalRepos(base)
	assert.Contains(t, repos, "real-repo-a")
	assert.Contains(t, repos, "real-repo-b")
	assert.NotContains(t, repos, "plain-dir")
	assert.NotContains(t, repos, "another-dir")
	assert.NotContains(t, repos, "some-file.txt")
	assert.Len(t, repos, 2)
}
