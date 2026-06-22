// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestBrainSeedMemoryCov_BrainSeedMemory_Bad_NoKey — with no brain API key the
// import is rejected up front.
func TestBrainSeedMemoryCov_BrainSeedMemory_Bad_NoKey(t *testing.T) {
	s := &PrepSubsystem{}
	result := s.brainSeedMemory(context.Background(), BrainSeedMemoryInput{
		WorkspaceID: 1,
		Path:        t.TempDir(),
	}, true)
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "no brain API key configured")
}

// TestBrainSeedMemoryCov_BrainSeedMemory_Good_DryRunCountsWithoutCalling — a
// dry run reports each section as imported without ever calling the brain API.
func TestBrainSeedMemoryCov_BrainSeedMemory_Good_DryRunCountsWithoutCalling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"),
		"# Memory\n\n## Architecture\nUse Core.Process().\n\n## Decision\nPrefer named actions.").OK)

	// No brainURL configured: a real call would fail, proving dry run never calls.
	s := &PrepSubsystem{brainKey: "brain-key"}

	var output BrainSeedMemoryOutput
	captureStdout(t, func() {
		result := s.brainSeedMemory(context.Background(), BrainSeedMemoryInput{
			WorkspaceID: 7,
			AgentID:     "virgil",
			Path:        memoryDir,
			DryRun:      true,
		}, true)
		core.RequireTrue(t, result.OK)
		var ok bool
		output, ok = result.Value.(BrainSeedMemoryOutput)
		core.RequireTrue(t, ok)
	})

	core.AssertEqual(t, 1, output.Files)
	core.AssertEqual(t, 2, output.Imported)
	core.AssertEqual(t, 0, output.Skipped)
	core.AssertTrue(t, output.DryRun)
}

// TestBrainSeedMemoryCov_BrainSeedMemory_Ugly_SkipsFileWithNoSections — a
// markdown file with no headings yields zero sections and is skipped.
func TestBrainSeedMemoryCov_BrainSeedMemory_Ugly_SkipsFileWithNoSections(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "just prose, no headings at all\n").OK)

	s := &PrepSubsystem{brainKey: "brain-key"}

	var output BrainSeedMemoryOutput
	captureStdout(t, func() {
		result := s.brainSeedMemory(context.Background(), BrainSeedMemoryInput{
			WorkspaceID: 7,
			AgentID:     "virgil",
			Path:        memoryDir,
		}, true)
		core.RequireTrue(t, result.OK)
		var ok bool
		output, ok = result.Value.(BrainSeedMemoryOutput)
		core.RequireTrue(t, ok)
	})

	core.AssertEqual(t, 1, output.Files)
	core.AssertEqual(t, 0, output.Imported)
	core.AssertEqual(t, 1, output.Skipped)
}

// TestBrainSeedMemoryCov_CmdBrainSeedMemory_Good_NoFilesFound — when the scan
// path has no MEMORY.md files the command prints the "no files" notice and
// returns OK with zero files.
func TestBrainSeedMemoryCov_CmdBrainSeedMemory_Good_NoFilesFound(t *testing.T) {
	empty := t.TempDir()
	s := &PrepSubsystem{brainURL: "https://example.com", brainKey: "brain-key"}

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdBrainSeedMemory(core.NewOptions(
			core.Option{Key: "workspace", Value: "1"},
			core.Option{Key: `path`, Value: empty},
		))
	})
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, output, "No markdown memory files found in:")

	out, ok := result.Value.(BrainSeedMemoryOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 0, out.Files)
}

// TestBrainSeedMemoryCov_CmdBrainSeedMemory_Bad_NoKeyError — a configured
// workspace but no brain key surfaces the brainSeedMemory error through the
// command's error-print path.
func TestBrainSeedMemoryCov_CmdBrainSeedMemory_Bad_NoKeyError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().").OK)

	s := &PrepSubsystem{} // no brainKey

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdBrainSeedMemory(core.NewOptions(
			core.Option{Key: "workspace", Value: "1"},
			core.Option{Key: `path`, Value: memoryDir},
		))
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
}

// TestBrainSeedMemoryCov_CmdBrainSeedMemory_Ugly_DryRunPrefix — a dry-run import
// prints the "[DRY RUN] Imported ..." summary line.
func TestBrainSeedMemoryCov_CmdBrainSeedMemory_Ugly_DryRunPrefix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().").OK)

	s := &PrepSubsystem{brainKey: "brain-key"}

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdBrainSeedMemory(core.NewOptions(
			core.Option{Key: "workspace", Value: "1"},
			core.Option{Key: `path`, Value: memoryDir},
			core.Option{Key: "dry-run", Value: true},
		))
	})
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, output, "[DRY RUN] Imported")
}

// TestBrainSeedMemoryCov_ExpandHome_Good_TildeAndPlain — a "~/..." path expands
// to the home dir; a plain path is returned unchanged.
func TestBrainSeedMemoryCov_ExpandHome_Good_TildeAndPlain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	core.AssertEqual(t, core.Concat(HomeDir(), "/notes/MEMORY.md"), brainSeedMemoryExpandHome("~/notes/MEMORY.md"))
	core.AssertEqual(t, "/abs/path", brainSeedMemoryExpandHome("/abs/path"))
}

// TestBrainSeedMemoryCov_ScanPath_Good_BlankFallsBackToDefault — a blank path
// falls back to the expanded default scan path.
func TestBrainSeedMemoryCov_ScanPath_Good_BlankFallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	scan := brainSeedMemoryScanPath("   ")
	core.AssertEqual(t, brainSeedMemoryExpandHome(brainSeedMemoryDefaultPath), scan)
}

// TestBrainSeedMemoryCov_Files_Good_AllMarkdownMode — in non-memory-only mode a
// directory walk collects every .md file (not just MEMORY.md), sorted.
func TestBrainSeedMemoryCov_Files_Good_AllMarkdownMode(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "a-notes.md"), "## H\nbody\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "MEMORY.md"), "## H\nbody\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "ignore.txt"), "x").OK)
	sub := core.JoinPath(dir, "nested")
	core.RequireTrue(t, fs.EnsureDir(sub).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(sub, "deep.md"), "## H\nbody\n").OK)

	files := brainSeedMemoryFiles(dir, false)
	core.AssertLen(t, files, 3)
	// Sorted ascending: a-notes.md < MEMORY.md (uppercase) is false; verify set.
	names := map[string]bool{}
	for _, f := range files {
		names[core.PathBase(f)] = true
	}
	core.AssertTrue(t, names["a-notes.md"])
	core.AssertTrue(t, names["MEMORY.md"])
	core.AssertTrue(t, names["deep.md"])
	core.AssertFalse(t, names["ignore.txt"])
}

// TestBrainSeedMemoryCov_Files_Bad_EmptyScanPath — a blank scan path returns nil.
func TestBrainSeedMemoryCov_Files_Bad_EmptyScanPath(t *testing.T) {
	core.AssertNil(t, brainSeedMemoryFiles("", true))
}

// TestBrainSeedMemoryCov_Files_Ugly_GlobMatchesDirectory — a glob whose matches
// are directories walks into each to collect the markdown files beneath.
func TestBrainSeedMemoryCov_Files_Ugly_GlobMatchesDirectory(t *testing.T) {
	root := t.TempDir()
	firstDir := core.JoinPath(root, "alpha", "memory")
	secondDir := core.JoinPath(root, "beta", "memory")
	core.RequireTrue(t, fs.EnsureDir(firstDir).OK)
	core.RequireTrue(t, fs.EnsureDir(secondDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(firstDir, "MEMORY.md"), "## H\nbody\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(secondDir, "MEMORY.md"), "## H\nbody\n").OK)

	// The glob "<root>/*/memory" matches two directories, each walked for files.
	files := brainSeedMemoryFiles(core.JoinPath(root, "*", "memory"), true)
	core.AssertLen(t, files, 2)
}

// TestBrainSeedMemoryCov_BrainSeedMemory_Ugly_SkipsOnImportFailure — when the
// brain API rejects every section, each is counted as skipped (not imported).
func TestBrainSeedMemoryCov_BrainSeedMemory_Ugly_SkipsOnImportFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"),
		"## Architecture\nUse Core.Process().\n\n## Decision\nPrefer named actions.").OK)

	srv := covMiscBrainAlwaysFailServer(t)
	s := &PrepSubsystem{brainURL: srv, brainKey: "brain-key"}

	var output BrainSeedMemoryOutput
	captureStdout(t, func() {
		result := s.brainSeedMemory(context.Background(), BrainSeedMemoryInput{
			WorkspaceID: 7,
			AgentID:     "virgil",
			Path:        memoryDir,
		}, true)
		core.RequireTrue(t, result.OK)
		var ok bool
		output, ok = result.Value.(BrainSeedMemoryOutput)
		core.RequireTrue(t, ok)
	})

	core.AssertEqual(t, 0, output.Imported)
	core.AssertEqual(t, 2, output.Skipped)
}

// TestBrainSeedMemoryCov_Type_Good_EachCategory — each keyword family resolves
// to its memory type, with the fallthrough returning "observation".
func TestBrainSeedMemoryCov_Type_Good_EachCategory(t *testing.T) {
	core.AssertEqual(t, "architecture", brainSeedMemoryType("Infrastructure", "the service mesh layer"))
	core.AssertEqual(t, "convention", brainSeedMemoryType("Naming standard", "coding pattern rule"))
	core.AssertEqual(t, "decision", brainSeedMemoryType("Strategy", "we chose this approach for the domain"))
	core.AssertEqual(t, "bug", brainSeedMemoryType("Lesson", "fix the broken error"))
	core.AssertEqual(t, "plan", brainSeedMemoryType("Roadmap", "phase milestone todo"))
	core.AssertEqual(t, "research", brainSeedMemoryType("RFC", "finding from analysis discovery"))
	core.AssertEqual(t, "observation", brainSeedMemoryType("Misc", "nothing classifiable here"))
}

// TestBrainSeedMemoryCov_Tags_Good_FilenameVariants — a hyphen/underscore
// filename becomes a spaced tag, a "memory" filename yields only the import tag,
// and an empty filename yields just the import tag.
func TestBrainSeedMemoryCov_Tags_Good_FilenameVariants(t *testing.T) {
	core.AssertEqual(t, []string{"project notes draft", "memory-import"}, brainSeedMemoryTags("project_notes-draft"))
	core.AssertEqual(t, []string{"memory-import"}, brainSeedMemoryTags("memory"))
	core.AssertEqual(t, []string{"memory-import"}, brainSeedMemoryTags(""))
}

// TestBrainSeedMemoryCov_Heading_Good_LevelBounds — only level 1-3 ATX headings
// with a space and text are accepted; level 4, no-space, and bare-hash lines are
// rejected.
func TestBrainSeedMemoryCov_Heading_Good_LevelBounds(t *testing.T) {
	h1, ok1 := brainSeedMemoryHeading("# Title")
	core.AssertTrue(t, ok1)
	core.AssertEqual(t, "Title", h1)

	_, ok4 := brainSeedMemoryHeading("#### Too deep")
	core.AssertFalse(t, ok4)

	_, okNoSpace := brainSeedMemoryHeading("##NoSpace")
	core.AssertFalse(t, okNoSpace)

	_, okBare := brainSeedMemoryHeading("###")
	core.AssertFalse(t, okBare)

	_, okPlain := brainSeedMemoryHeading("not a heading")
	core.AssertFalse(t, okPlain)
}

// TestBrainSeedMemoryCov_Project_Bad_NoMemorySegment — a path with no "memory"
// segment yields an empty project string.
func TestBrainSeedMemoryCov_Project_Bad_NoMemorySegment(t *testing.T) {
	core.AssertEqual(t, "", brainSeedMemoryProject("/Users/snider/notes/file.md"))
}

// covMiscBrainAlwaysFailServer starts an httptest server that rejects every
// request with 400 (a non-retryable status), so brainCall fails fast.
func covMiscBrainAlwaysFailServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rejected", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}
