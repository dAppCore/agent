// SPDX-License-Identifier: EUPL-1.2

package lib

import (
	"embed"
	"runtime"
	"testing"

	core "dappco.re/go/core"
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
		if !testFs.Exists(core.JoinPath(dir, name)) {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestLib_ExtractWorkspaceSubdirs_Good(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "test task"}

	requireExtractWorkspaceOK(t, ExtractWorkspace("default", dir, data))

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
	requireExtractWorkspaceError(t, ExtractWorkspace("missing-template", t.TempDir(), &WorkspaceData{Repo: "test-repo"}))
}

func TestLib_ExtractWorkspace_Ugly(t *testing.T) {
	requireExtractWorkspaceError(t, ExtractWorkspace("default", t.TempDir(), nil))
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

func TestLib_ReferenceFiles_Good_SPDXHeaders(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	repoRoot := core.PathDir(core.PathDir(core.PathDir(file)))
	refDir := core.JoinPath(repoRoot, ".core", "reference")
	goFiles := core.PathGlob(core.JoinPath(refDir, "*.go"))
	if len(goFiles) == 0 {
		t.Fatalf("no .go files found in %s", refDir)
	}

	for _, path := range goFiles {
		assertSPDXHeader(t, path)
	}
}

func TestLib_ExtractWorkspace_Good_ReferenceHeaders(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "carry SPDX headers into workspace references"}

	requireExtractWorkspaceOK(t, ExtractWorkspace("default", dir, data))

	refDir := core.JoinPath(dir, ".core", "reference")
	goFiles := core.PathGlob(core.JoinPath(refDir, "*.go"))
	if len(goFiles) == 0 {
		t.Fatalf("no extracted .go files found in %s", refDir)
	}

	for _, path := range goFiles {
		assertSPDXHeader(t, path)
	}
}

func TestLib_ExtractWorkspace_Good_ReferenceUsageExamples(t *testing.T) {
	dir := t.TempDir()
	data := &WorkspaceData{Repo: "test-repo", Task: "carry AX usage examples into workspace references"}

	requireExtractWorkspaceOK(t, ExtractWorkspace("default", dir, data))

	cases := map[string][]string{
		core.JoinPath(dir, ".core", "reference", "array.go"): {
			`arr := core.NewArray("prep", "dispatch")`,
			`arr.Add("verify", "merge")`,
			`arr.AddUnique("verify", "verify", "merge")`,
		},
		core.JoinPath(dir, ".core", "reference", "config.go"): {
			`timeout := core.ConfigGet[int](c.Config(), "agent.timeout")`,
		},
		core.JoinPath(dir, ".core", "reference", "embed.go"): {
			`core.AddAsset("docs", "RFC.md", packed)`,
			`r := core.GeneratePack(pkg)`,
		},
		core.JoinPath(dir, ".core", "reference", "error.go"): {
			`if core.Is(err, context.Canceled) { return }`,
			`stack := core.FormatStackTrace(err)`,
			`r := c.Error().Reports(10)`,
		},
		core.JoinPath(dir, ".core", "reference", "log.go"): {
			`log := core.NewLog(core.LogOptions{Level: core.LevelDebug, Output: os.Stdout})`,
			`core.SetRedactKeys("token", "password")`,
			`core.Security("entitlement.denied", "action", "process.run")`,
		},
		core.JoinPath(dir, ".core", "reference", "runtime.go"): {
			`r := c.ServiceStartup(context.Background(), nil)`,
			`r := core.NewRuntime(app)`,
			`name := runtime.ServiceName()`,
		},
	}

	for path, snippets := range cases {
		r := testFs.Read(path)
		if !r.OK {
			t.Fatalf("failed to read %s", path)
		}
		text := r.Value.(string)
		for _, snippet := range snippets {
			if !core.Contains(text, snippet) {
				t.Errorf("%s missing usage example snippet %q", path, snippet)
			}
		}
	}
}

func TestLib_MountEmbed_Bad(t *testing.T) {
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
