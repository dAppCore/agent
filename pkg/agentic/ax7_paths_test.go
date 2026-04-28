// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func ax7WorkspaceDir(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-5")
	core.RequireTrue(t, fs.EnsureDir(WorkspaceRepoDir(wsDir)).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(wsDir)).OK)
	return wsDir
}

func ax7WriteStatus(t *testing.T, workspaceDir string) {
	t.Helper()

	core.RequireTrue(t, fs.Write(WorkspaceStatusPath(workspaceDir), core.JSONMarshalString(&WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "AX7",
	})).OK)
}

func TestCoreHome_HomeDir_Good(t *testing.T) {
	t.Setenv("CORE_HOME", "/tmp/core-home")
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("DIR_HOME", "/tmp/dir-home")

	got := HomeDir()
	core.AssertEqual(t, "/tmp/core-home", got)
	core.AssertContains(t, got, "core-home")
}

func TestHomeFallback_HomeDir_Bad(t *testing.T) {
	t.Setenv("CORE_HOME", "")
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("DIR_HOME", "/tmp/dir-home")

	got := HomeDir()
	core.AssertEqual(t, "/tmp/home", got)
	core.AssertContains(t, got, "home")
}

func TestDirHomeFallback_HomeDir_Ugly(t *testing.T) {
	t.Setenv("CORE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("DIR_HOME", "/tmp/dir-home")

	got := HomeDir()
	core.AssertEqual(t, "/tmp/dir-home", got)
	core.AssertContains(t, got, "dir-home")
}

func TestWorkspaceEnv_CoreRoot_Good(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	got := CoreRoot()
	core.AssertEqual(t, "/tmp/test-core", got)
	core.AssertContains(t, got, "test-core")
}

func TestWhitespaceWorkspace_CoreRoot_Bad(t *testing.T) {
	setTestWorkspace(t, "   ")
	got := CoreRoot()
	core.AssertEqual(t, "   ", got)
	core.AssertNotEmpty(t, got)
}

func TestUnicodeWorkspace_CoreRoot_Ugly(t *testing.T) {
	setTestWorkspace(t, "/tmp/\u00e9\u00e0\u00fc")
	got := CoreRoot()
	core.AssertEqual(t, "/tmp/\u00e9\u00e0\u00fc", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestWorkspaceEnv_WorkspaceRoot_Good(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	got := WorkspaceRoot()
	core.AssertEqual(t, "/tmp/test-core/workspace", got)
	core.AssertContains(t, got, "workspace")
}

func TestFallbackRoot_WorkspaceRoot_Bad(t *testing.T) {
	setTestWorkspace(t, "")
	got := WorkspaceRoot()
	core.AssertContains(t, got, "/Code/.core/workspace")
	core.AssertContains(t, got, "workspace")
}

func TestTrailingSlash_WorkspaceRoot_Ugly(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core/")
	got := WorkspaceRoot()
	core.AssertNotEmpty(t, got)
	core.AssertContains(t, got, "workspace")
}

func TestWorkspaceEnv_PlansRoot_Good(t *testing.T) {
	setTestWorkspace(t, "/tmp/test-core")
	got := PlansRoot()
	core.AssertEqual(t, "/tmp/test-core/plans", got)
	core.AssertContains(t, got, "plans")
}

func TestFallbackRoot_PlansRoot_Bad(t *testing.T) {
	setTestWorkspace(t, "")
	got := PlansRoot()
	core.AssertContains(t, got, "/Code/.core/plans")
	core.AssertContains(t, got, "plans")
}

func TestNestedRoot_PlansRoot_Ugly(t *testing.T) {
	setTestWorkspace(t, "/a/b/c/d/e/f")
	got := PlansRoot()
	core.AssertEqual(t, "/a/b/c/d/e/f/plans", got)
	core.AssertContains(t, got, "/plans")
}

func TestExplicitAgent_AgentName_Good(t *testing.T) {
	t.Setenv("AGENT_NAME", "clotho")
	got := AgentName()
	core.AssertEqual(t, "clotho", got)
	core.AssertNotEmpty(t, got)
}

func TestWhitespaceAgent_AgentName_Bad(t *testing.T) {
	t.Setenv("AGENT_NAME", "   ")
	got := AgentName()
	core.AssertEqual(t, "   ", got)
	core.AssertNotEmpty(t, got)
}

func TestUnicodeAgent_AgentName_Ugly(t *testing.T) {
	t.Setenv("AGENT_NAME", "\u00e9nchantr\u00efx")
	got := AgentName()
	core.AssertEqual(t, "\u00e9nchantr\u00efx", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestExplicitOrg_GitHubOrg_Good(t *testing.T) {
	t.Setenv("GITHUB_ORG", "myorg")
	got := GitHubOrg()
	core.AssertEqual(t, "myorg", got)
	core.AssertNotEmpty(t, got)
}

func TestWhitespaceOrg_GitHubOrg_Bad(t *testing.T) {
	t.Setenv("GITHUB_ORG", "   ")
	got := GitHubOrg()
	core.AssertEqual(t, "   ", got)
	core.AssertNotEmpty(t, got)
}

func TestSpecialCharsOrg_GitHubOrg_Ugly(t *testing.T) {
	t.Setenv("GITHUB_ORG", "org/with/slashes")
	got := GitHubOrg()
	core.AssertEqual(t, "org/with/slashes", got)
	core.AssertContains(t, got, "/")
}

func TestUnrestrictedFs_LocalFs_Good(t *testing.T) {
	got := LocalFs()
	core.AssertNotNil(t, got)
	assertIsType(t, &core.Fs{}, got)
}

func TestMissingFile_LocalFs_Bad(t *testing.T) {
	result := LocalFs().Read("/tmp/nonexistent-path-agentic-ax7/file.txt")
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestEmptyPath_LocalFs_Ugly(t *testing.T) {
	core.AssertNotPanics(t, func() {
		LocalFs().Read("")
	})
	core.AssertNotNil(t, LocalFs())
}

func TestWorkspaceStatusFile_WorkspaceStatusPath_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceStatusPath(wsDir)
	core.AssertEqual(t, core.JoinPath(wsDir, "status.json"), got)
	core.AssertContains(t, got, "status.json")
}

func TestEmptyDir_WorkspaceStatusPath_Bad(t *testing.T) {
	got := WorkspaceStatusPath("")
	core.AssertEqual(t, "/status.json", got)
	core.AssertContains(t, got, "status.json")
}

func TestUnicodeDir_WorkspaceStatusPath_Ugly(t *testing.T) {
	got := WorkspaceStatusPath("/tmp/\u00e9")
	core.AssertEqual(t, "/tmp/\u00e9/status.json", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestStandardName_WorkspaceName_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceName(wsDir)
	core.AssertEqual(t, "core/go-io/task-5", got)
	core.AssertContains(t, got, "go-io")
}

func TestRootFallback_WorkspaceName_Bad(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	got := WorkspaceName(WorkspaceRoot())
	core.AssertEqual(t, "workspace", got)
	core.AssertContains(t, got, "workspace")
}

func TestSlashName_WorkspaceName_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "feature", "new-ui")
	got := WorkspaceName(wsDir)
	core.AssertEqual(t, "core/go-io/feature/new-ui", got)
	core.AssertContains(t, got, "feature")
}

func TestRepoPath_WorkspaceRepoDir_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceRepoDir(wsDir)
	core.AssertEqual(t, core.JoinPath(wsDir, "repo"), got)
	core.AssertContains(t, got, "/repo")
}

func TestEmptyDir_WorkspaceRepoDir_Bad(t *testing.T) {
	got := WorkspaceRepoDir("")
	core.AssertEqual(t, "/repo", got)
	core.AssertContains(t, got, "repo")
}

func TestUnicodeDir_WorkspaceRepoDir_Ugly(t *testing.T) {
	got := WorkspaceRepoDir("/tmp/\u00e9")
	core.AssertEqual(t, "/tmp/\u00e9/repo", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestMetaPath_WorkspaceMetaDir_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceMetaDir(wsDir)
	core.AssertEqual(t, core.JoinPath(wsDir, ".meta"), got)
	core.AssertContains(t, got, ".meta")
}

func TestEmptyDir_WorkspaceMetaDir_Bad(t *testing.T) {
	got := WorkspaceMetaDir("")
	core.AssertEqual(t, "/.meta", got)
	core.AssertContains(t, got, ".meta")
}

func TestUnicodeDir_WorkspaceMetaDir_Ugly(t *testing.T) {
	got := WorkspaceMetaDir("/tmp/\u00e9")
	core.AssertEqual(t, "/tmp/\u00e9/.meta", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestBlockedPath_WorkspaceBlockedPath_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceBlockedPath(wsDir)
	core.AssertEqual(t, core.JoinPath(wsDir, "repo", "BLOCKED.md"), got)
	core.AssertContains(t, got, "BLOCKED.md")
}

func TestEmptyDir_WorkspaceBlockedPath_Bad(t *testing.T) {
	got := WorkspaceBlockedPath("")
	core.AssertEqual(t, "/repo/BLOCKED.md", got)
	core.AssertContains(t, got, "BLOCKED.md")
}

func TestUnicodeDir_WorkspaceBlockedPath_Ugly(t *testing.T) {
	got := WorkspaceBlockedPath("/tmp/\u00e9")
	core.AssertEqual(t, "/tmp/\u00e9/repo/BLOCKED.md", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestAnswerPath_WorkspaceAnswerPath_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceAnswerPath(wsDir)
	core.AssertEqual(t, core.JoinPath(wsDir, "repo", "ANSWER.md"), got)
	core.AssertContains(t, got, "ANSWER.md")
}

func TestEmptyDir_WorkspaceAnswerPath_Bad(t *testing.T) {
	got := WorkspaceAnswerPath("")
	core.AssertEqual(t, "/repo/ANSWER.md", got)
	core.AssertContains(t, got, "ANSWER.md")
}

func TestUnicodeDir_WorkspaceAnswerPath_Ugly(t *testing.T) {
	got := WorkspaceAnswerPath("/tmp/\u00e9")
	core.AssertEqual(t, "/tmp/\u00e9/repo/ANSWER.md", got)
	core.AssertContains(t, got, "\u00e9")
}

func TestExistingLogs_WorkspaceLogFiles_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	logPath := core.JoinPath(WorkspaceMetaDir(wsDir), "agent-codex.log")
	core.RequireTrue(t, fs.Write(logPath, "done").OK)

	got := WorkspaceLogFiles(wsDir)
	core.AssertContains(t, got, logPath)
	core.AssertLen(t, got, 1)
}

func TestMissingLogs_WorkspaceLogFiles_Bad(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	got := WorkspaceLogFiles(wsDir)
	core.AssertEmpty(t, got)
	core.AssertLen(t, got, 0)
}

func TestMultipleLogs_WorkspaceLogFiles_Ugly(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	first := core.JoinPath(WorkspaceMetaDir(wsDir), "agent-codex.log")
	second := core.JoinPath(WorkspaceMetaDir(wsDir), "agent-claude.log")
	core.RequireTrue(t, fs.Write(first, "done").OK)
	core.RequireTrue(t, fs.Write(second, "done").OK)

	got := WorkspaceLogFiles(wsDir)
	core.AssertContains(t, got, first)
	core.AssertContains(t, got, second)
}

func TestWorkspaceTree_WorkspaceStatusPaths_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-5")
	core.RequireTrue(t, fs.EnsureDir(WorkspaceRepoDir(wsDir)).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(wsDir)).OK)
	ax7WriteStatus(t, wsDir)

	got := WorkspaceStatusPaths()
	core.AssertContains(t, got, WorkspaceStatusPath(wsDir))
	core.AssertLen(t, got, 1)
}

func TestEmptyTree_WorkspaceStatusPaths_Bad(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	got := WorkspaceStatusPaths()
	core.AssertEmpty(t, got)
	core.AssertLen(t, got, 0)
}

func TestDeepTree_WorkspaceStatusPaths_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	shallow := core.JoinPath(WorkspaceRoot(), "ws-flat")
	deep := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-12")
	core.RequireTrue(t, fs.EnsureDir(shallow).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceRepoDir(deep)).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(deep)).OK)
	core.RequireTrue(t, fs.Write(WorkspaceStatusPath(shallow), "{}").OK)
	ax7WriteStatus(t, deep)

	got := WorkspaceStatusPaths()
	core.AssertContains(t, got, WorkspaceStatusPath(shallow))
	core.AssertContains(t, got, WorkspaceStatusPath(deep))
}

func TestStatusFile_ReadStatusResult_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	ax7WriteStatus(t, wsDir)

	result := ReadStatusResult(wsDir)
	core.RequireTrue(t, result.OK)
	status, ok := result.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "running", status.Status)
}

func TestMissingStatus_ReadStatusResult_Bad(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestInvalidJSON_ReadStatusResult_Ugly(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	core.RequireTrue(t, fs.Write(WorkspaceStatusPath(wsDir), "{not-json").OK)

	result := ReadStatusResult(wsDir)
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestStatusFile_ReadStatus_Good(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	ax7WriteStatus(t, wsDir)

	status, err := ReadStatus(wsDir)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "running", status.Status)
}

func TestMissingStatus_ReadStatus_Bad(t *testing.T) {
	status, err := ReadStatus(t.TempDir())
	core.AssertNil(t, status)
	core.AssertError(t, err)
}

func TestInvalidJSON_ReadStatus_Ugly(t *testing.T) {
	wsDir := ax7WorkspaceDir(t)
	core.RequireTrue(t, fs.Write(WorkspaceStatusPath(wsDir), "{not-json").OK)

	status, err := ReadStatus(wsDir)
	core.AssertNil(t, status)
	core.AssertError(t, err)
}
