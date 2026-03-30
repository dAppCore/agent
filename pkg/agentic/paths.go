// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strconv"

	core "dappco.re/go/core"
)

// fs provides unrestricted filesystem access (root "/" = no sandbox).
//
//	r := fs.Read("/etc/hostname")
//	if r.OK { core.Print(nil, "%s", r.Value.(string)) }
var fs = (&core.Fs{}).NewUnrestricted()

// LocalFs returns an unrestricted filesystem instance for use by other packages.
//
//	f := agentic.LocalFs()
//	r := f.Read("/tmp/agent-status.json")
func LocalFs() *core.Fs { return fs }

// WorkspaceRoot returns the root directory for agent workspaces.
// Checks CORE_WORKSPACE env var first, falls back to HomeDir()/Code/.core/workspace.
//
//	wsDir := core.JoinPath(agentic.WorkspaceRoot(), "core", "go-io", "task-42")
func WorkspaceRoot() string {
	return core.JoinPath(CoreRoot(), "workspace")
}

// WorkspaceStatusPaths returns all workspace status files across supported layouts.
//
//	paths := agentic.WorkspaceStatusPaths()
func WorkspaceStatusPaths() []string {
	return workspaceStatusPaths(WorkspaceRoot())
}

// WorkspaceStatusPath returns the status file for a workspace directory.
//
//	path := agentic.WorkspaceStatusPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceStatusPath(wsDir string) string {
	return core.JoinPath(wsDir, "status.json")
}

// WorkspaceName extracts the unique workspace name from a full path.
// Given /Users/snider/Code/.core/workspace/core/go-io/dev → core/go-io/dev
//
//	name := agentic.WorkspaceName("/Users/snider/Code/.core/workspace/core/go-io/dev")
func WorkspaceName(wsDir string) string {
	root := WorkspaceRoot()
	name := core.TrimPrefix(wsDir, root)
	name = core.TrimPrefix(name, "/")
	if name == "" {
		return core.PathBase(wsDir)
	}
	return name
}

// CoreRoot returns the root directory for core ecosystem files.
// Checks CORE_WORKSPACE env var first, falls back to HomeDir()/Code/.core.
//
//	root := agentic.CoreRoot()
func CoreRoot() string {
	if root := core.Env("CORE_WORKSPACE"); root != "" {
		return root
	}
	return core.JoinPath(HomeDir(), "Code", ".core")
}

// HomeDir returns the user home directory used by agentic path helpers.
//
//	home := agentic.HomeDir()
func HomeDir() string {
	if home := core.Env("CORE_HOME"); home != "" {
		return home
	}
	if home := core.Env("HOME"); home != "" {
		return home
	}
	return core.Env("DIR_HOME")
}

func workspaceStatusPaths(wsRoot string) []string {
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))
	return append(old, deep...)
}

// WorkspaceRepoDir returns the checked-out repo directory for a workspace.
//
//	repoDir := agentic.WorkspaceRepoDir("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceRepoDir(wsDir string) string {
	return core.JoinPath(wsDir, "repo")
}

func workspaceRepoDir(wsDir string) string {
	return WorkspaceRepoDir(wsDir)
}

// WorkspaceMetaDir returns the metadata directory for a workspace.
//
//	metaDir := agentic.WorkspaceMetaDir("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceMetaDir(wsDir string) string {
	return core.JoinPath(wsDir, ".meta")
}

func workspaceMetaDir(wsDir string) string {
	return WorkspaceMetaDir(wsDir)
}

// WorkspaceBlockedPath returns the BLOCKED.md path for a workspace.
//
//	blocked := agentic.WorkspaceBlockedPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceBlockedPath(wsDir string) string {
	return core.JoinPath(WorkspaceRepoDir(wsDir), "BLOCKED.md")
}

func workspaceBlockedPath(wsDir string) string {
	return WorkspaceBlockedPath(wsDir)
}

// WorkspaceAnswerPath returns the ANSWER.md path for a workspace.
//
//	answer := agentic.WorkspaceAnswerPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceAnswerPath(wsDir string) string {
	return core.JoinPath(WorkspaceRepoDir(wsDir), "ANSWER.md")
}

func workspaceAnswerPath(wsDir string) string {
	return WorkspaceAnswerPath(wsDir)
}

// WorkspaceLogFiles returns captured agent log files for a workspace.
//
//	logs := agentic.WorkspaceLogFiles("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceLogFiles(wsDir string) []string {
	return core.PathGlob(core.JoinPath(WorkspaceMetaDir(wsDir), "agent-*.log"))
}

func workspaceLogFiles(wsDir string) []string {
	return WorkspaceLogFiles(wsDir)
}

// PlansRoot returns the root directory for agent plans.
//
//	plansDir := agentic.PlansRoot()
func PlansRoot() string {
	return core.JoinPath(CoreRoot(), "plans")
}

// AgentName returns the name of this agent based on hostname.
// Checks AGENT_NAME env var first.
//
//	name := agentic.AgentName() // "cladius" on Snider's Mac, "charon" elsewhere
func AgentName() string {
	if name := core.Env("AGENT_NAME"); name != "" {
		return name
	}
	h := core.Lower(core.Env("HOSTNAME"))
	if core.Contains(h, "snider") || core.Contains(h, "studio") || core.Contains(h, "mac") {
		return "cladius"
	}
	return "charon"
}

// DefaultBranch detects the default branch of a repo (main, master, etc.).
//
//	base := s.DefaultBranch("./src")
func (s *PrepSubsystem) DefaultBranch(repoDir string) string {
	ctx := context.Background()
	process := s.Core().Process()
	if r := process.RunIn(ctx, repoDir, "git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short"); r.OK {
		ref := core.Trim(r.Value.(string))
		if core.HasPrefix(ref, "origin/") {
			return core.TrimPrefix(ref, "origin/")
		}
		return ref
	}
	for _, branch := range []string{"main", "master"} {
		if process.RunIn(ctx, repoDir, "git", "rev-parse", "--verify", branch).OK {
			return branch
		}
	}
	return "main"
}

// GitHubOrg returns the GitHub org for mirror operations.
//
//	org := agentic.GitHubOrg() // "dAppCore"
func GitHubOrg() string {
	if org := core.Env("GITHUB_ORG"); org != "" {
		return org
	}
	return "dAppCore"
}

func parseInt(value string) int {
	n, err := strconv.Atoi(core.Trim(value))
	if err != nil {
		return 0
	}
	return n
}
