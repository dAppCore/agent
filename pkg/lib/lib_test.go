package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractWorkspace_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	// Check top-level template files exist
	for _, name := range []string{"CODEX.md", "CLAUDE.md", "PROMPT.md", "TODO.md", "CONTEXT.md"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestExtractWorkspace_CreatesSubdirectories(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	// Check .core/reference/ directory exists with files
	refDir := filepath.Join(dir, ".core", "reference")
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Fatalf(".core/reference/ directory not created")
	}

	// Check AX spec exists
	axSpec := filepath.Join(refDir, "RFC-025-AGENT-EXPERIENCE.md")
	if _, err := os.Stat(axSpec); os.IsNotExist(err) {
		t.Errorf("AX spec not extracted: %s", axSpec)
	}

	// Check Core source files exist
	entries, err := os.ReadDir(refDir)
	if err != nil {
		t.Fatalf("failed to read reference dir: %v", err)
	}

	goFiles := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".go" {
			goFiles++
		}
	}
	if goFiles == 0 {
		t.Error("no .go files in .core/reference/")
	}

	// Check docs subdirectory
	docsDir := filepath.Join(refDir, "docs")
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		t.Errorf(".core/reference/docs/ not created")
	}
}

func TestExtractWorkspace_TemplateSubstitution(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "my-repo", Task: "fix the bug"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	// TODO.md should contain the task
	content, err := os.ReadFile(filepath.Join(dir, "TODO.md"))
	if err != nil {
		t.Fatalf("failed to read TODO.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("TODO.md is empty")
	}
}
