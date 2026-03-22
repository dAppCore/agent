package lib

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// --- Prompt ---

func TestPrompt_Good(t *testing.T) {
	r := Prompt("coding")
	if !r.OK {
		t.Fatal("Prompt('coding') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Prompt('coding') returned empty string")
	}
}

func TestPrompt_Bad(t *testing.T) {
	r := Prompt("nonexistent-slug")
	if r.OK {
		t.Error("Prompt('nonexistent-slug') should return !OK")
	}
}

// --- Task ---

func TestTask_Good_Yaml(t *testing.T) {
	r := Task("bug-fix")
	if !r.OK {
		t.Fatal("Task('bug-fix') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Task('bug-fix') returned empty string")
	}
}

func TestTask_Good_Md(t *testing.T) {
	r := Task("code/review")
	if !r.OK {
		t.Fatal("Task('code/review') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Task('code/review') returned empty string")
	}
}

func TestTask_Bad(t *testing.T) {
	r := Task("nonexistent-slug")
	if r.OK {
		t.Error("Task('nonexistent-slug') should return !OK")
	}
	if r.Value != fs.ErrNotExist {
		t.Error("Task('nonexistent-slug') should return fs.ErrNotExist")
	}
}

// --- TaskBundle ---

func TestTaskBundle_Good(t *testing.T) {
	r := TaskBundle("code/review")
	if !r.OK {
		t.Fatal("TaskBundle('code/review') returned !OK")
	}
	b := r.Value.(Bundle)
	if b.Main == "" {
		t.Error("Bundle.Main is empty")
	}
	if len(b.Files) == 0 {
		t.Error("Bundle.Files is empty — expected companion files")
	}
}

func TestTaskBundle_Bad(t *testing.T) {
	r := TaskBundle("nonexistent")
	if r.OK {
		t.Error("TaskBundle('nonexistent') should return !OK")
	}
}

// --- Flow ---

func TestFlow_Good(t *testing.T) {
	r := Flow("go")
	if !r.OK {
		t.Fatal("Flow('go') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Flow('go') returned empty string")
	}
}

// --- Persona ---

func TestPersona_Good(t *testing.T) {
	// Use first persona from list to avoid hardcoding
	personas := ListPersonas()
	if len(personas) == 0 {
		t.Skip("no personas found")
	}
	r := Persona(personas[0])
	if !r.OK {
		t.Fatalf("Persona(%q) returned !OK", personas[0])
	}
	if r.Value.(string) == "" {
		t.Errorf("Persona(%q) returned empty string", personas[0])
	}
}

// --- Template ---

func TestTemplate_Good_Prompt(t *testing.T) {
	r := Template("coding")
	if !r.OK {
		t.Fatal("Template('coding') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Template('coding') returned empty string")
	}
}

func TestTemplate_Good_TaskFallback(t *testing.T) {
	r := Template("bug-fix")
	if !r.OK {
		t.Fatal("Template('bug-fix') returned !OK — should fall through to Task")
	}
}

func TestTemplate_Bad(t *testing.T) {
	r := Template("nonexistent-slug")
	if r.OK {
		t.Error("Template('nonexistent-slug') should return !OK")
	}
}

// --- List Functions ---

func TestListPrompts(t *testing.T) {
	prompts := ListPrompts()
	if len(prompts) == 0 {
		t.Error("ListPrompts() returned empty")
	}
}

func TestListTasks(t *testing.T) {
	tasks := ListTasks()
	if len(tasks) == 0 {
		t.Fatal("ListTasks() returned empty")
	}
	// Verify nested paths are included (e.g., "code/review")
	found := false
	for _, s := range tasks {
		if s == "code/review" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListTasks() missing nested path 'code/review'")
	}
}

func TestListPersonas(t *testing.T) {
	personas := ListPersonas()
	if len(personas) == 0 {
		t.Error("ListPersonas() returned empty")
	}
	// Should have nested paths like "code/go"
	hasNested := false
	for _, p := range personas {
		if len(p) > 0 && filepath.Dir(p) != "." {
			hasNested = true
			break
		}
	}
	if !hasNested {
		t.Error("ListPersonas() has no nested paths")
	}
}

func TestListFlows(t *testing.T) {
	flows := ListFlows()
	if len(flows) == 0 {
		t.Error("ListFlows() returned empty")
	}
}

func TestListWorkspaces(t *testing.T) {
	workspaces := ListWorkspaces()
	if len(workspaces) == 0 {
		t.Error("ListWorkspaces() returned empty")
	}
}

// --- ExtractWorkspace ---

func TestExtractWorkspace_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	for _, name := range []string{"CODEX.md", "CLAUDE.md", "PROMPT.md", "TODO.md", "CONTEXT.md", "go.work"} {
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

	refDir := filepath.Join(dir, ".core", "reference")
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Fatalf(".core/reference/ directory not created")
	}

	axSpec := filepath.Join(refDir, "RFC-025-AGENT-EXPERIENCE.md")
	if _, err := os.Stat(axSpec); os.IsNotExist(err) {
		t.Errorf("AX spec not extracted: %s", axSpec)
	}

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

	content, err := os.ReadFile(filepath.Join(dir, "TODO.md"))
	if err != nil {
		t.Fatalf("failed to read TODO.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("TODO.md is empty")
	}
}
