// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleWorkspaceRoot() {
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}

func ExampleLocalFs() {
	core.Println(LocalFs() != nil)
	// Output: true
}

func ExampleCoreRoot() {
	root := CoreRoot()
	core.Println(core.HasSuffix(root, ".core"))
	// Output: true
}

func ExampleHomeDir() {
	home := HomeDir()
	core.Println(home != "")
	// Output: true
}

func ExamplePlansRoot() {
	root := PlansRoot()
	core.Println(core.HasSuffix(root, "plans"))
	// Output: true
}

func ExampleAgentName() {
	name := AgentName()
	core.Println(name != "")
	// Output: true
}

func ExampleGitHubOrg() {
	org := GitHubOrg()
	core.Println(org != "")
	// Output: true
}

func ExampleWorkspaceStatusPaths() {
	fsys := (&core.Fs{}).NewUnrestricted()
	root := "/tmp/core-agent-paths-example"
	defer fsys.DeleteAll(root)
	setWorkspaceRootOverride(root)
	defer setWorkspaceRootOverride("")
	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-1")
	_ = fs.EnsureDir(WorkspaceRepoDir(workspaceDir))
	_ = fs.Write(WorkspaceStatusPath(workspaceDir), "{}")
	core.Println(len(WorkspaceStatusPaths()))
	// Output: 1
}

func ExampleWorkspaceStatusPath() {
	core.Println(WorkspaceStatusPath("/tmp/workspace"))
	// Output: /tmp/workspace/status.json
}

func ExampleWorkspaceName() {
	setWorkspaceRootOverride("/tmp/core-agent-paths-example")
	defer setWorkspaceRootOverride("")
	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-2")
	core.Println(WorkspaceName(workspaceDir))
	// Output: core/go-io/task-2
}

func ExampleWorkspaceRepoDir() {
	core.Println(WorkspaceRepoDir("/tmp/workspace"))
	// Output: /tmp/workspace/repo
}

func ExampleWorkspaceMetaDir() {
	core.Println(WorkspaceMetaDir("/tmp/workspace"))
	// Output: /tmp/workspace/.meta
}

func ExampleWorkspaceBlockedPath() {
	core.Println(WorkspaceBlockedPath("/tmp/workspace"))
	// Output: /tmp/workspace/repo/BLOCKED.md
}

func ExampleWorkspaceAnswerPath() {
	core.Println(WorkspaceAnswerPath("/tmp/workspace"))
	// Output: /tmp/workspace/repo/ANSWER.md
}

func ExampleWorkspaceLogFiles() {
	fsys := (&core.Fs{}).NewUnrestricted()
	root := "/tmp/core-agent-logfiles-example"
	defer fsys.DeleteAll(root)
	_ = fs.EnsureDir(WorkspaceMetaDir(root))
	_ = fs.Write(core.JoinPath(WorkspaceMetaDir(root), "agent-codex.log"), "done")
	core.Println(len(WorkspaceLogFiles(root)))
	// Output: 1
}

func ExamplePrepSubsystem_DefaultBranch() {
	fsys := (&core.Fs{}).NewUnrestricted()
	repoDir := "/tmp/core-agent-default-branch"
	defer fsys.DeleteAll(repoDir)
	_ = fs.EnsureDir(repoDir)
	subsystem := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{})}
	core.Println(subsystem.DefaultBranch(repoDir))
	// Output: main
}
