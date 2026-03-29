// SPDX-License-Identifier: EUPL-1.2

package lib

import (
	"testing"

	core "dappco.re/go/core"
)

var testFs = (&core.Fs{}).NewUnrestricted()

// --- Prompt ---

func TestLib_Prompt_Good(t *testing.T) {
	r := Prompt("coding")
	if !r.OK {
		t.Fatal("Prompt('coding') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Prompt('coding') returned empty string")
	}
}

func TestLib_Prompt_Bad(t *testing.T) {
	r := Prompt("nonexistent-slug")
	if r.OK {
		t.Error("Prompt('nonexistent-slug') should return !OK")
	}
}

// --- Task ---

func TestLib_Task_Good(t *testing.T) {
	r := Task("bug-fix")
	if !r.OK {
		t.Fatal("Task('bug-fix') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Task('bug-fix') returned empty string")
	}
}

func TestLib_TaskNested_Good(t *testing.T) {
	r := Task("code/review")
	if !r.OK {
		t.Fatal("Task('code/review') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Task('code/review') returned empty string")
	}
}

func TestLib_Task_Bad(t *testing.T) {
	r := Task("nonexistent-slug")
	if r.OK {
		t.Error("Task('nonexistent-slug') should return !OK")
	}
}

// --- TaskBundle ---

func TestLib_TaskBundle_Good(t *testing.T) {
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

func TestLib_TaskBundle_Bad(t *testing.T) {
	r := TaskBundle("nonexistent")
	if r.OK {
		t.Error("TaskBundle('nonexistent') should return !OK")
	}
}

// --- Flow ---

func TestLib_Flow_Good(t *testing.T) {
	r := Flow("go")
	if !r.OK {
		t.Fatal("Flow('go') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Flow('go') returned empty string")
	}
}

// --- Persona ---

func TestLib_Persona_Good(t *testing.T) {
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

func TestLib_Template_Good(t *testing.T) {
	r := Template("coding")
	if !r.OK {
		t.Fatal("Template('coding') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("Template('coding') returned empty string")
	}
}

func TestLib_TemplateFallback_Good(t *testing.T) {
	r := Template("bug-fix")
	if !r.OK {
		t.Fatal("Template('bug-fix') returned !OK — should fall through to Task")
	}
}

func TestLib_Template_Bad(t *testing.T) {
	r := Template("nonexistent-slug")
	if r.OK {
		t.Error("Template('nonexistent-slug') should return !OK")
	}
}

// --- List Functions ---

func TestLib_ListPrompts_Good(t *testing.T) {
	prompts := ListPrompts()
	if len(prompts) == 0 {
		t.Error("ListPrompts() returned empty")
	}
}

func TestLib_ListTasks_Good(t *testing.T) {
	tasks := ListTasks()
	if len(tasks) == 0 {
		t.Fatal("ListTasks() returned empty")
	}
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

func TestLib_ListPersonas_Good(t *testing.T) {
	personas := ListPersonas()
	if len(personas) == 0 {
		t.Error("ListPersonas() returned empty")
	}
	hasNested := false
	for _, p := range personas {
		if len(p) > 0 && core.PathDir(p) != "." {
			hasNested = true
			break
		}
	}
	if !hasNested {
		t.Error("ListPersonas() has no nested paths")
	}
}

func TestLib_ListFlows_Good(t *testing.T) {
	flows := ListFlows()
	if len(flows) == 0 {
		t.Error("ListFlows() returned empty")
	}
}

func TestLib_ListWorkspaces_Good(t *testing.T) {
	workspaces := ListWorkspaces()
	if len(workspaces) == 0 {
		t.Error("ListWorkspaces() returned empty")
	}
}

// --- ExtractWorkspace ---

func TestLib_ExtractWorkspace_Good(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	for _, name := range []string{"CODEX.md", "CLAUDE.md", "PROMPT.md", "TODO.md", "CONTEXT.md", "go.work"} {
		if !testFs.Exists(core.JoinPath(dir, name)) {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestLib_ExtractWorkspaceSubdirs_Good(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	refDir := core.JoinPath(dir, ".core", "reference")
	if !testFs.IsDir(refDir) {
		t.Fatalf(".core/reference/ directory not created")
	}

	axSpec := core.JoinPath(refDir, "RFC-025-AGENT-EXPERIENCE.md")
	if !testFs.Exists(axSpec) {
		t.Errorf("AX spec not extracted: %s", axSpec)
	}

	goFiles := core.PathGlob(core.JoinPath(refDir, "*.go"))
	if len(goFiles) == 0 {
		t.Error("no .go files in .core/reference/")
	}

	docsDir := core.JoinPath(refDir, "docs")
	if !testFs.IsDir(docsDir) {
		t.Errorf(".core/reference/docs/ not created")
	}
}

func TestLib_ExtractWorkspaceTemplate_Good(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "my-repo", Task: "fix the bug"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	r := testFs.Read(core.JoinPath(dir, "TODO.md"))
	if !r.OK {
		t.Fatalf("failed to read TODO.md")
	}
	if r.Value.(string) == "" {
		t.Error("TODO.md is empty")
	}
}

func TestLib_ExtractWorkspace_Good_AXConventions(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "align AX docs"}

	err := ExtractWorkspace("default", dir, data)
	if err != nil {
		t.Fatalf("ExtractWorkspace failed: %v", err)
	}

	r := testFs.Read(core.JoinPath(dir, "CODEX.md"))
	if !r.OK {
		t.Fatalf("failed to read CODEX.md")
	}

	text := r.Value.(string)
	for _, banned := range []string{
		"c.PERFORM(",
		"c.RegisterTask(",
		"OnStartup(ctx context.Context) error",
		"OnShutdown(ctx context.Context) error",
	} {
		if core.Contains(text, banned) {
			t.Errorf("CODEX.md still contains deprecated AX guidance: %s", banned)
		}
	}

	for _, required := range []string{
		"core.WithService(",
		"c.Action(\"workspace.create\"",
		"c.Task(\"deploy\"",
		"c.Process().RunIn(",
		"TestFile_Function_Good",
	} {
		if !core.Contains(text, required) {
			t.Errorf("CODEX.md missing AX guidance: %s", required)
		}
	}
}
