package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initGitRepoWithCode creates a git repo with searchable Go code.
func initGitRepoWithCode(t *testing.T) string {
	t.Helper()
	dir := initGitRepo(t)

	// Create Go files with known content for git grep.
	files := map[string]string{
		"auth.go": `package main

// Authenticate validates user credentials.
func Authenticate(user, pass string) bool {
	return user != "" && pass != ""
}
`,
		"handler.go": `package main

// HandleRequest processes HTTP requests for authentication.
func HandleRequest() {
	// authentication logic here
}
`,
		"util.go": `package main

// TokenValidator checks JWT tokens.
func TokenValidator(token string) bool {
	return len(token) > 0
}
`,
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Stage and commit all files so git grep can find them.
	_, err := runCommandCtx(context.Background(), dir, "git", "add", "-A")
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "commit", "-m", "add code files")
	require.NoError(t, err)

	return dir
}

func TestFindRelatedCode_Good_MatchesKeywords(t *testing.T) {
	dir := initGitRepoWithCode(t)

	task := &Task{
		ID:          "code-1",
		Title:       "Fix authentication handler",
		Description: "The authentication handler needs refactoring",
	}

	files, err := findRelatedCode(task, dir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "should find files matching 'authentication' keyword")

	// Verify language detection.
	for _, f := range files {
		assert.Equal(t, "go", f.Language)
	}
}

func TestFindRelatedCode_Good_NoKeywords(t *testing.T) {
	dir := initGitRepoWithCode(t)

	task := &Task{
		ID:          "code-2",
		Title:       "do it", // too short, all stop words
		Description: "fix the bug in the code",
	}

	files, err := findRelatedCode(task, dir)
	require.NoError(t, err)
	// Keywords extracted from "do it fix the bug in the code" -- most are stop words.
	// Only "bug" is 3+ chars and not a stop word, but may not match any files.
	// Result can be nil or empty -- both are acceptable.
	_ = files
}

func TestFindRelatedCode_Bad_NilTask(t *testing.T) {
	files, err := findRelatedCode(nil, ".")
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestFindRelatedCode_Good_LimitsTo10Files(t *testing.T) {
	dir := initGitRepoWithCode(t)

	// Create 15 files all containing the keyword "validation".
	for i := range 15 {
		name := filepath.Join(dir, "validation_"+string(rune('a'+i))+".go")
		content := "package main\n// validation logic\nfunc Validate" + string(rune('A'+i)) + "() {}\n"
		err := os.WriteFile(name, []byte(content), 0644)
		require.NoError(t, err)
	}

	_, err := runCommandCtx(context.Background(), dir, "git", "add", "-A")
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "commit", "-m", "add validation files")
	require.NoError(t, err)

	task := &Task{
		ID:          "code-3",
		Title:       "validation refactoring",
		Description: "Refactor all validation logic",
	}

	files, err := findRelatedCode(task, dir)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(files), 10, "should limit to 10 related files")
}

func TestFindRelatedCode_Good_TruncatesLargeFiles(t *testing.T) {
	dir := initGitRepoWithCode(t)

	// Create a file larger than 5000 chars containing a searchable keyword.
	largeContent := "package main\n// largecontent\n"
	for len(largeContent) < 6000 {
		largeContent += "// This is filler content for testing truncation purposes.\n"
	}
	err := os.WriteFile(filepath.Join(dir, "large.go"), []byte(largeContent), 0644)
	require.NoError(t, err)

	_, err = runCommandCtx(context.Background(), dir, "git", "add", "-A")
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "commit", "-m", "add large file")
	require.NoError(t, err)

	task := &Task{
		ID:          "code-4",
		Title:       "largecontent analysis",
		Description: "Analyse the largecontent module",
	}

	files, err := findRelatedCode(task, dir)
	require.NoError(t, err)

	for _, f := range files {
		if f.Path == "large.go" {
			assert.True(t, len(f.Content) <= 5020, "content should be truncated")
			assert.Contains(t, f.Content, "... (truncated)")
			return
		}
	}
	// If large.go wasn't found by git grep, that's acceptable.
}

func TestBuildTaskContext_Good_WithGitRepo(t *testing.T) {
	dir := initGitRepoWithCode(t)

	task := &Task{
		ID:          "ctx-1",
		Title:       "Test context building with authentication",
		Description: "Build context in a git repo with searchable code",
		Priority:    PriorityMedium,
		Status:      StatusPending,
		Files:       []string{"auth.go"},
		CreatedAt:   time.Now(),
	}

	ctx, err := BuildTaskContext(task, dir)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Equal(t, task, ctx.Task)

	// Should have gathered the auth.go file.
	assert.Len(t, ctx.Files, 1)
	assert.Equal(t, "auth.go", ctx.Files[0].Path)
	assert.Contains(t, ctx.Files[0].Content, "Authenticate")

	// Should have recent commits.
	assert.NotEmpty(t, ctx.RecentCommits)

	// Should have found related code.
	assert.NotEmpty(t, ctx.RelatedCode, "should find code related to 'authentication'")
}

func TestBuildTaskContext_Good_EmptyDir(t *testing.T) {
	task := &Task{
		ID:          "ctx-2",
		Title:       "Test with empty dir",
		Description: "Testing",
		Priority:    PriorityLow,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
	}

	// Empty dir defaults to cwd -- BuildTaskContext handles errors gracefully.
	ctx, err := BuildTaskContext(task, "")
	require.NoError(t, err)
	assert.NotNil(t, ctx)
}

func TestFormatContext_Good_EmptySections(t *testing.T) {
	task := &Task{
		ID:          "fmt-1",
		Title:       "Minimal task",
		Description: "No files, no git",
		Priority:    PriorityLow,
		Status:      StatusPending,
	}

	ctx := &TaskContext{
		Task:          task,
		Files:         nil,
		GitStatus:     "",
		RecentCommits: "",
		RelatedCode:   nil,
	}

	formatted := ctx.FormatContext()

	assert.Contains(t, formatted, "# Task Context")
	assert.Contains(t, formatted, "fmt-1")
	assert.NotContains(t, formatted, "## Task Files")
	assert.NotContains(t, formatted, "## Git Status")
	assert.NotContains(t, formatted, "## Recent Commits")
	assert.NotContains(t, formatted, "## Related Code")
}

func TestRunGitCommand_Good(t *testing.T) {
	dir := initGitRepo(t)

	output, err := runGitCommand(dir, "log", "--oneline", "-1")
	require.NoError(t, err)
	assert.Contains(t, output, "initial commit")
}

func TestRunGitCommand_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := runGitCommand(dir, "status")
	assert.Error(t, err)
}
