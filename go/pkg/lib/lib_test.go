// SPDX-License-Identifier: EUPL-1.2

package lib

import (
	"embed"
	"slices"
	"testing"

	core "dappco.re/go"
)

var testFs = (&core.Fs{}).NewUnrestricted()

func breakLibMountForTest(t *testing.T) {
	t.Helper()

	originalPromptFiles := promptFiles
	promptFiles = embed.FS{}
	mountDone.Store(false)
	mountResult = core.Result{}
	data = nil
	promptFS = nil
	taskFS = nil
	flowFS = nil
	personaFS = nil
	workspaceFS = nil

	t.Cleanup(func() {
		promptFiles = originalPromptFiles
		mountDone.Store(false)
		mountResult = core.Result{}
		data = nil
		promptFS = nil
		taskFS = nil
		flowFS = nil
		personaFS = nil
		workspaceFS = nil
	})
}

func corruptLibMountForTest(t *testing.T) {
	t.Helper()

	MountData(core.New())
	data = nil

	t.Cleanup(func() {
		mountDone.Store(false)
		mountResult = core.Result{}
		data = nil
	})
}

func requireExtractWorkspaceOK(t *testing.T, result core.Result) string {
	t.Helper()
	if !result.OK {
		t.Fatalf("ExtractWorkspace failed: %v", result.Value)
	}
	path, ok := result.Value.(string)
	if !ok {
		t.Fatalf("ExtractWorkspace returned %T, want string", result.Value)
	}
	return path
}

func requireExtractWorkspaceError(t *testing.T, result core.Result) error {
	t.Helper()
	if result.OK {
		t.Fatalf("ExtractWorkspace unexpectedly succeeded: %#v", result.Value)
	}
	err, ok := result.Value.(error)
	if !ok {
		t.Fatalf("ExtractWorkspace returned %T, want error", result.Value)
	}
	return err
}

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

func TestLib_Prompt_Ugly(t *testing.T) {
	r := Prompt("../coding")
	if r.OK {
		t.Error("Prompt('../coding') should return !OK")
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

func TestLib_TaskNested_Good_Case(t *testing.T) {
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

func TestLib_Task_Ugly(t *testing.T) {
	r := Task("../bug-fix")
	if r.OK {
		t.Error("Task('../bug-fix') should return !OK")
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

func TestLib_TaskBundle_Ugly(t *testing.T) {
	r := TaskBundle("../code/review")
	if r.OK {
		t.Error("TaskBundle('../code/review') should return !OK")
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

func TestLib_Flow_Bad(t *testing.T) {
	r := Flow("nonexistent-flow")
	if r.OK {
		t.Error("Flow('nonexistent-flow') should return !OK")
	}
}

func TestLib_Flow_Ugly(t *testing.T) {
	r := Flow("../go")
	if r.OK {
		t.Error("Flow('../go') should return !OK")
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

func TestLib_Persona_Bad(t *testing.T) {
	r := Persona("nonexistent-persona")
	if r.OK {
		t.Error("Persona('nonexistent-persona') should return !OK")
	}
}

func TestLib_Persona_Ugly(t *testing.T) {
	r := Persona("../secops/developer")
	if r.OK {
		t.Error("Persona('../secops/developer') should return !OK")
	}
}

// --- PersonaCards ---

func TestLib_PersonaCards_Good(t *testing.T) {
	cards := PersonaCards()
	if len(cards) == 0 {
		t.Fatal("PersonaCards() returned no cards")
	}
	// The starting roster is present and named from its frontmatter.
	want := map[string]string{
		"code/senior-developer": "Senior Developer",
		"code/technical-writer": "Technical Writer",
		"secops/developer":      "Security Developer",
		"testing/tester":        "Tester",
	}
	seen := map[string]string{}
	for _, c := range cards {
		if name, ok := want[c.Path]; ok {
			seen[c.Path] = c.Name
			if c.Name != name {
				t.Errorf("card %q: Name = %q, want %q", c.Path, c.Name, name)
			}
		}
	}
	for path := range want {
		if _, ok := seen[path]; !ok {
			t.Errorf("starting-roster persona %q missing from PersonaCards()", path)
		}
	}
}

func TestLib_PersonaCards_Bad(t *testing.T) {
	// Filter invariant: a returned card always carries a dispatch path and a
	// frontmatter name — files without frontmatter (docs, playbooks) are
	// dropped, never returned blank.
	for _, c := range PersonaCards() {
		if c.Path == "" || c.Name == "" {
			t.Errorf("PersonaCards() returned an incomplete card: %+v", c)
		}
	}
}

func TestLib_PersonaCards_Ugly(t *testing.T) {
	// The recursive persona walk surfaces directory entries too; PersonaCards
	// must filter them — fewer cards than raw paths, and never a bare dir.
	cards := PersonaCards()
	if len(cards) >= len(ListPersonas()) {
		t.Errorf("PersonaCards (%d) should be fewer than raw ListPersonas (%d) — dirs/docs unfiltered",
			len(cards), len(ListPersonas()))
	}
	for _, c := range cards {
		switch c.Path {
		case "code", "secops", "testing", "design", "devops", "plan", "product":
			t.Errorf("PersonaCards() leaked a directory entry: %q", c.Path)
		}
	}
}

// --- TaskCards ---

func TestLib_TaskCards_Good(t *testing.T) {
	cards := TaskCards()
	if len(cards) == 0 {
		t.Fatal("TaskCards() returned no cards")
	}
	// The premade-task staples are present and named from their yaml.
	want := map[string]string{
		"package-update":   "Package Update",
		"dependency-audit": "Dependency Audit",
	}
	seen := map[string]bool{}
	for _, c := range cards {
		if name, ok := want[c.Slug]; ok {
			seen[c.Slug] = true
			if c.Name != name {
				t.Errorf("card %q: Name = %q, want %q", c.Slug, c.Name, name)
			}
		}
	}
	for slug := range want {
		if !seen[slug] {
			t.Errorf("task template %q missing from TaskCards()", slug)
		}
	}
}

func TestLib_TaskCards_Bad(t *testing.T) {
	// Every returned card carries a slug and a name — directory entries and
	// nameless files are filtered, never returned blank.
	for _, c := range TaskCards() {
		if c.Slug == "" || c.Name == "" {
			t.Errorf("TaskCards() returned an incomplete card: %+v", c)
		}
	}
}

func TestLib_TaskCards_Ugly(t *testing.T) {
	// The recursive task walk surfaces directory entries (e.g. "code");
	// TaskCards must filter them — fewer cards than raw slugs, none a dir.
	cards := TaskCards()
	if len(cards) >= len(ListTasks()) {
		t.Errorf("TaskCards (%d) should be fewer than raw ListTasks (%d) — dirs unfiltered",
			len(cards), len(ListTasks()))
	}
	for _, c := range cards {
		if c.Slug == "code" {
			t.Errorf("TaskCards() leaked a directory entry: %q", c.Slug)
		}
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

func TestLib_TemplateFallback_Good_Case(t *testing.T) {
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

func TestLib_Template_Ugly(t *testing.T) {
	r := Template("../coding")
	if r.OK {
		t.Error("Template('../coding') should return !OK")
	}
}

// --- WorkspaceFile ---

func TestLib_WorkspaceFile_Good(t *testing.T) {
	r := WorkspaceFile("default", "CODEX.md.tmpl")
	if !r.OK {
		t.Fatal("WorkspaceFile('default', 'CODEX.md.tmpl') returned !OK")
	}
	if r.Value.(string) == "" {
		t.Error("WorkspaceFile('default', 'CODEX.md.tmpl') returned empty string")
	}
}

func TestLib_WorkspaceFile_Bad(t *testing.T) {
	r := WorkspaceFile("missing-template", "CODEX.md.tmpl")
	if r.OK {
		t.Error("WorkspaceFile('missing-template', 'CODEX.md.tmpl') should return !OK")
	}
}

func TestLib_WorkspaceFile_Ugly(t *testing.T) {
	r := WorkspaceFile("default", "../CODEX.md.tmpl")
	if r.OK {
		t.Error("WorkspaceFile('default', '../CODEX.md.tmpl') should return !OK")
	}
}

// --- MountData ---

func TestLib_MountData_Good(t *testing.T) {
	c := core.New()
	MountData(c)

	r := c.Data().ReadString("prompts/coding.md")
	if !r.OK {
		t.Fatal("MountData() did not register prompt data")
	}
}

func TestLib_MountData_Bad(t *testing.T) {
	breakLibMountForTest(t)

	c := core.New()
	MountData(c)

	r := c.Data().ReadString("prompts/coding.md")
	if r.OK {
		t.Error("MountData() should not register prompt data when mounting fails")
	}
}

func TestLib_MountData_Ugly(t *testing.T) {
	corruptLibMountForTest(t)

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		MountData(nil)
	}()
	if !panicked {
		t.Fatal("MountData(nil) should panic")
	}
}

// --- List Functions ---

func TestLib_ListPrompts_Good(t *testing.T) {
	prompts := ListPrompts()
	if len(prompts) == 0 {
		t.Error("ListPrompts() returned empty")
	}
}

func TestLib_ListPrompts_Bad(t *testing.T) {
	breakLibMountForTest(t)

	if prompts := ListPrompts(); prompts != nil {
		t.Error("ListPrompts() should return nil when mounting fails")
	}
}

func TestLib_ListPrompts_Ugly(t *testing.T) {
	corruptLibMountForTest(t)

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = ListPrompts()
	}()
	if !panicked {
		t.Fatal("ListPrompts() should panic when embedded state is corrupted")
	}
}

func TestLib_ListTasks_Good(t *testing.T) {
	tasks := ListTasks()
	if len(tasks) == 0 {
		t.Fatal("ListTasks() returned empty")
	}
	found := slices.Contains(tasks, "code/review")
	if !found {
		t.Error("ListTasks() missing nested path 'code/review'")
	}
}

func TestLib_ListTasks_Bad(t *testing.T) {
	breakLibMountForTest(t)

	if tasks := ListTasks(); tasks != nil {
		t.Error("ListTasks() should return nil when mounting fails")
	}
}

func TestLib_ListTasks_Ugly(t *testing.T) {
	corruptLibMountForTest(t)

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = ListTasks()
	}()
	if !panicked {
		t.Fatal("ListTasks() should panic when embedded state is corrupted")
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

func TestLib_ListPersonas_Bad(t *testing.T) {
	breakLibMountForTest(t)

	if personas := ListPersonas(); personas != nil {
		t.Error("ListPersonas() should return nil when mounting fails")
	}
}

func TestLib_ListPersonas_Ugly(t *testing.T) {
	corruptLibMountForTest(t)

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = ListPersonas()
	}()
	if !panicked {
		t.Fatal("ListPersonas() should panic when embedded state is corrupted")
	}
}

func TestLib_ListFlows_Good(t *testing.T) {
	flows := ListFlows()
	if len(flows) == 0 {
		t.Error("ListFlows() returned empty")
	}
}

func TestLib_ListFlows_Bad(t *testing.T) {
	breakLibMountForTest(t)

	if flows := ListFlows(); flows != nil {
		t.Error("ListFlows() should return nil when mounting fails")
	}
}

func TestLib_ListFlows_Ugly(t *testing.T) {
	corruptLibMountForTest(t)

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = ListFlows()
	}()
	if !panicked {
		t.Fatal("ListFlows() should panic when embedded state is corrupted")
	}
}

func TestLib_ListWorkspaces_Good(t *testing.T) {
	workspaces := ListWorkspaces()
	if len(workspaces) == 0 {
		t.Error("ListWorkspaces() returned empty")
	}
}

func TestLib_ListWorkspaces_Bad(t *testing.T) {
	breakLibMountForTest(t)

	if workspaces := ListWorkspaces(); workspaces != nil {
		t.Error("ListWorkspaces() should return nil when mounting fails")
	}
}

func TestLib_ListWorkspaces_Ugly(t *testing.T) {
	corruptLibMountForTest(t)

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = ListWorkspaces()
	}()
	if !panicked {
		t.Fatal("ListWorkspaces() should panic when embedded state is corrupted")
	}
}

// --- ExtractWorkspace ---

func TestLib_ExtractWorkspace_Good(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	requireExtractWorkspaceOK(t, ExtractWorkspace("default", dir, data))

	for _, name := range []string{"CODEX.md", "CLAUDE.md", "PROMPT.md", "TODO.md", "CONTEXT.md", "go.work"} {
		if !testFs.Exists(core.JoinPath(dir, name)).OK {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestLib_ExtractWorkspaceTemplate_Good_Case(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "my-repo", Task: "fix the bug"}

	requireExtractWorkspaceOK(t, ExtractWorkspace("default", dir, data))

	r := testFs.Read(core.JoinPath(dir, "TODO.md"))
	if !r.OK {
		t.Fatalf("failed to read TODO.md")
	}
	if r.Value.(string) == "" {
		t.Error("TODO.md is empty")
	}
}

func TestLib_ExtractWorkspace_Bad(t *testing.T) {
	result := ExtractWorkspace("missing-template", t.TempDir(), &WorkspaceData{Repo: "test-repo"})
	requireExtractWorkspaceError(t, result)
	core.AssertFalse(t, result.OK)
}

func TestLib_ExtractWorkspace_Ugly(t *testing.T) {
	result := ExtractWorkspace("default", t.TempDir(), nil)
	requireExtractWorkspaceError(t, result)
	core.AssertFalse(t, result.OK)
}

func TestLib_ExtractWorkspace_Good_AXConventions(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "align AX docs"}

	requireExtractWorkspaceOK(t, ExtractWorkspace("default", dir, data))

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

func TestLib_MountEmbed_Bad_Case(t *testing.T) {
	result := mountEmbed(promptFiles, "missing-dir")
	if result.OK {
		t.Fatal("mountEmbed should fail for a missing embedded directory")
	}
	if _, ok := result.Value.(error); !ok {
		t.Fatal("mountEmbed should return an error value")
	}
}

func assertSPDXHeader(t *testing.T, path string) {
	t.Helper()

	r := testFs.Read(path)
	if !r.OK {
		t.Fatalf("failed to read %s", path)
	}
	if !core.HasPrefix(r.Value.(string), "// SPDX-License-Identifier: EUPL-1.2") {
		t.Fatalf("%s missing SPDX header", path)
	}
}
