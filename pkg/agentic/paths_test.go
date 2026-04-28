// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strings"
	"testing"

	core "dappco.re/go"
)

func TestPaths_CoreRoot_Good_EnvVar(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	core.AssertEqual(
		t,
		"/tmp/test-core",
		CoreRoot(),
	)
}

func TestPaths_CoreRoot_Good_Fallback(t *testing.T) {
	setTestWorkspace(t, "")
	home := HomeDir()
	core.AssertEqual(t, home+"/Code/.core", CoreRoot())
}

func TestPaths_CoreRoot_Good_CoreHome(t *testing.T) {
	setTestWorkspace(t, "")
	t.Setenv("CORE_HOME", "/tmp/core-home")
	core.AssertEqual(t, "/tmp/core-home/Code/.core", CoreRoot())
}

func TestPaths_HomeDir_Good_CoreHome(t *testing.T) {
	t.Setenv("CORE_HOME", "/tmp/core-home")
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("DIR_HOME", "/tmp/dir-home")
	core.AssertEqual(t, "/tmp/core-home", HomeDir())
}

func TestPaths_HomeDir_Good_HomeFallback(t *testing.T) {
	t.Setenv("CORE_HOME", "")
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("DIR_HOME", "/tmp/dir-home")
	core.AssertEqual(t, "/tmp/home", HomeDir())
}

func TestPaths_WorkspaceRoot_Good(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	core.AssertEqual(
		t,
		"/tmp/test-core/workspace",
		WorkspaceRoot(),
	)
}

func TestPaths_WorkspaceHelpers_Good(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	wsDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-5")
	metaDir := WorkspaceMetaDir(wsDir)

	core.AssertEqual(t, core.JoinPath(wsDir, "status.json"), WorkspaceStatusPath(wsDir))
	core.AssertEqual(t, core.JoinPath(wsDir, "repo"), WorkspaceRepoDir(wsDir))
	core.AssertEqual(t, core.JoinPath(wsDir, ".meta"), metaDir)
	core.AssertEqual(t, core.JoinPath(wsDir, "repo", "BLOCKED.md"), WorkspaceBlockedPath(wsDir))
	core.AssertEqual(t, core.JoinPath(wsDir, "repo", "ANSWER.md"), WorkspaceAnswerPath(wsDir))
	core.AssertEqual(t, "core/go-io/task-5", WorkspaceName(wsDir))

	core.AssertTrue(t, fs.EnsureDir(metaDir).OK)
	core.AssertTrue(t, fs.Write(core.JoinPath(metaDir, "agent-codex.log"), "done").OK)
	core.AssertContains(t, WorkspaceLogFiles(wsDir), core.JoinPath(metaDir, "agent-codex.log"))
}

func TestPaths_WorkspaceHelpers_Good_BranchNameWithSlash(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	wsDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "feature", "new-ui")

	core.RequireTrue(t, fs.EnsureDir(WorkspaceRepoDir(wsDir)).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(wsDir)).OK)
	core.RequireTrue(t, fs.Write(WorkspaceStatusPath(wsDir), "{}").OK)

	core.AssertEqual(t, "core/go-io/feature/new-ui", WorkspaceName(wsDir))
	core.AssertContains(t, WorkspaceStatusPaths(), WorkspaceStatusPath(wsDir))
}

func TestPaths_PlansRoot_Good(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	core.AssertEqual(
		t,
		"/tmp/test-core/plans",
		PlansRoot(),
	)
}

func TestPaths_AgentName_Good_EnvVar(t *testing.T) {
	t.Setenv("AGENT_NAME", "clotho")
	core.AssertEqual(
		t,
		"clotho",
		AgentName(),
	)
}

func TestPaths_AgentName_Good_Fallback(t *testing.T) {
	t.Setenv("AGENT_NAME", "")
	name := AgentName()
	core.AssertTrue(t, name == "cladius" || name == "charon", "expected cladius or charon, got %s", name)
}

func TestPaths_GitHubOrg_Good_EnvVar(t *testing.T) {
	t.Setenv("GITHUB_ORG", "myorg")
	core.AssertEqual(
		t,
		"myorg",
		GitHubOrg(),
	)
}

func TestPaths_GitHubOrg_Good_Fallback(t *testing.T) {
	t.Setenv("GITHUB_ORG", "")
	core.AssertEqual(
		t,
		"dAppCore",
		GitHubOrg(),
	)
}

// --- DefaultBranch ---

func TestPaths_DefaultBranch_Good(t *testing.T) {
	dir := t.TempDir()

	// Init git repo with "main" branch
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", dir)
	testCore.Process().RunIn(context.Background(), dir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), dir, "git", "config", "user.email", "test@test.com")

	fs.Write(dir+"/README.md", "# Test")
	testCore.Process().RunIn(context.Background(), dir, "git", "add", ".")
	testCore.Process().RunIn(context.Background(), dir, "git", "commit", "-m", "init")

	branch := testPrep.DefaultBranch(dir)
	core.AssertEqual(t, "main", branch)
}

func TestPaths_DefaultBranch_Bad(t *testing.T) {
	// Non-git directory — should return "main" (default)
	dir := t.TempDir()
	branch := testPrep.DefaultBranch(dir)
	core.AssertEqual(t, "main", branch)
}

func TestDefaultBranchMaster_PrepSubsystem_DefaultBranch_Ugly(t *testing.T) {
	dir := t.TempDir()

	// Init git repo with "master" branch
	testCore.Process().Run(context.Background(), "git", "init", "-b", "master", dir)
	testCore.Process().RunIn(context.Background(), dir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), dir, "git", "config", "user.email", "test@test.com")

	fs.Write(dir+"/README.md", "# Test")
	testCore.Process().RunIn(context.Background(), dir, "git", "add", ".")
	testCore.Process().RunIn(context.Background(), dir, "git", "commit", "-m", "init")

	branch := testPrep.DefaultBranch(dir)
	core.AssertEqual(t, "master", branch)
}

// --- LocalFs Bad/Ugly ---

func TestPaths_LocalFs_Bad_ReadNonExistent(t *testing.T) {
	f := LocalFs()
	r := f.Read("/tmp/nonexistent-path-" + strings.Repeat("x", 20) + "/file.txt")
	core.AssertFalse(t, r.OK, "reading a non-existent file should fail")
}

func TestPaths_LocalFs_Ugly_EmptyPath(t *testing.T) {
	f := LocalFs()
	core.AssertNotPanics(t, func() {
		f.Read("")
	})
}

// --- WorkspaceRoot Bad/Ugly ---

func TestPaths_WorkspaceRoot_Bad_EmptyEnv(t *testing.T) {
	setTestWorkspace(t, "")
	home := HomeDir()
	// Should fall back to ~/Code/.core/workspace
	core.AssertEqual(t, home+"/Code/.core/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceHelpers_Bad(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	core.AssertEqual(t, "/status.json", WorkspaceStatusPath(""))
	core.AssertEqual(t, "/repo", WorkspaceRepoDir(""))
	core.AssertEqual(t, "/.meta", WorkspaceMetaDir(""))
	core.AssertEqual(t, "workspace", WorkspaceName(WorkspaceRoot()))
	core.AssertEmpty(t, WorkspaceLogFiles("/tmp/missing-workspace"))
}

func TestPaths_WorkspaceRoot_Ugly_TrailingSlash(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core/")
	// Verify it still constructs a valid path (JoinPath handles trailing slash)
	ws := WorkspaceRoot()
	core.AssertNotEmpty(t, ws)
	core.AssertContains(t, ws, "workspace")
}

func TestPaths_WorkspaceHelpers_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := WorkspaceRoot()

	shallow := core.JoinPath(wsRoot, "ws-flat")
	deep := core.JoinPath(wsRoot, "core", "go-io", "task-12")
	ignored := core.JoinPath(wsRoot, "core", "go-io", "task-12", "extra")

	core.AssertTrue(t, fs.EnsureDir(shallow).OK)
	core.AssertTrue(t, fs.EnsureDir(deep).OK)
	core.AssertTrue(t, fs.EnsureDir(ignored).OK)
	core.AssertTrue(t, fs.Write(core.JoinPath(shallow, "status.json"), "{}").OK)
	core.AssertTrue(t, fs.Write(core.JoinPath(deep, "status.json"), "{}").OK)
	core.AssertTrue(t, fs.Write(core.JoinPath(ignored, "status.json"), "{}").OK)

	paths := WorkspaceStatusPaths()
	core.AssertContains(t, paths, core.JoinPath(shallow, "status.json"))
	core.AssertContains(t, paths, core.JoinPath(deep, "status.json"))
	core.AssertNotContains(t, paths, core.JoinPath(ignored, "status.json"))
}

// --- CoreRoot Bad/Ugly ---

func TestPaths_CoreRoot_Bad_WhitespaceEnv(t *testing.T) {
	setTestWorkspace(t, "   ")
	// Non-empty string (whitespace) will be used as-is
	root := CoreRoot()
	core.AssertEqual(t, "   ", root)
}

func TestPaths_CoreRoot_Ugly_UnicodeEnv(t *testing.T) {
	setTestWorkspace(t, "/tmp/\u00e9\u00e0\u00fc")
	core.AssertNotPanics(t, func() {
		root := CoreRoot()
		core.AssertEqual(t, "/tmp/\u00e9\u00e0\u00fc", root)
	})
}

// --- PlansRoot Bad/Ugly ---

func TestPaths_PlansRoot_Bad_EmptyEnv(t *testing.T) {
	setTestWorkspace(t, "")
	home := HomeDir()
	core.AssertEqual(t, home+"/Code/.core/plans", PlansRoot())
}

func TestPaths_PlansRoot_Ugly_NestedPath(t *testing.T) {
	setTestWorkspace(t, "/a/b/c/d/e/f")
	core.AssertEqual(
		t,
		"/a/b/c/d/e/f/plans",
		PlansRoot(),
	)
}

// --- AgentName Bad/Ugly ---

func TestPaths_AgentName_Bad_WhitespaceEnv(t *testing.T) {
	t.Setenv("AGENT_NAME", "   ")
	// Whitespace is non-empty, so returned as-is
	core.AssertEqual(
		t,
		"   ",
		AgentName(),
	)
}

func TestPaths_AgentName_Ugly_UnicodeEnv(t *testing.T) {
	t.Setenv("AGENT_NAME", "\u00e9nchantr\u00efx")
	core.AssertNotPanics(t, func() {
		name := AgentName()
		core.AssertEqual(t, "\u00e9nchantr\u00efx", name)
	})
}

// --- GitHubOrg Bad/Ugly ---

func TestPaths_GitHubOrg_Bad_WhitespaceEnv(t *testing.T) {
	t.Setenv("GITHUB_ORG", "   ")
	core.AssertEqual(
		t,
		"   ",
		GitHubOrg(),
	)
}

func TestPaths_GitHubOrg_Ugly_SpecialChars(t *testing.T) {
	t.Setenv("GITHUB_ORG", "org/with/slashes")
	core.AssertNotPanics(t, func() {
		org := GitHubOrg()
		core.AssertEqual(t, "org/with/slashes", org)
	})
}

// --- parseInt Bad/Ugly ---

func TestPaths_ParseInt_Bad_EmptyString(t *testing.T) {
	core.AssertEqual(
		t,
		0,
		parseInt(""),
	)
}

func TestPaths_ParseInt_Bad_NonNumeric(t *testing.T) {
	core.AssertEqual(t, 0, parseInt("abc"))
	core.AssertEqual(t, 0, parseInt("12.5"))
	core.AssertEqual(t, 0, parseInt("0xff"))
}

func TestPaths_ParseInt_Bad_WhitespaceOnly(t *testing.T) {
	core.AssertEqual(
		t,
		0,
		parseInt("   "),
	)
}

func TestPaths_ParseInt_Ugly_NegativeNumber(t *testing.T) {
	core.AssertEqual(
		t,
		-42,
		parseInt("-42"),
	)
}

func TestPaths_ParseInt_Ugly_VeryLargeNumber(t *testing.T) {
	core.AssertEqual(
		t,
		0,
		parseInt("99999999999999999999999"),
	)
}

func TestPaths_ParseInt_Ugly_LeadingTrailingWhitespace(t *testing.T) {
	core.AssertEqual(
		t,
		42,
		parseInt("  42  "),
	)
}

// --- fs (NewUnrestricted) Good ---

func TestPaths_Fs_Good_Unrestricted(t *testing.T) {
	core.AssertNotNil(
		t,
		fs,
		"package-level fs should be non-nil",
	)
	assertIsType(t, &core.Fs{}, fs)
}

// --- parseInt Good ---

func TestPaths_ParseInt_Good(t *testing.T) {
	first := parseInt("42")
	second := parseInt("0")
	core.AssertEqual(t, 42, first)
	core.AssertEqual(t, 0, second)
}
