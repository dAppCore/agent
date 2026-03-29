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
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core/workspace.
//
//	wsDir := core.JoinPath(agentic.WorkspaceRoot(), "go-io-1774149757")
func WorkspaceRoot() string {
	return core.JoinPath(CoreRoot(), "workspace")
}

// WorkspaceStatusPaths returns all workspace status files across supported layouts.
//
//	paths := agentic.WorkspaceStatusPaths()
func WorkspaceStatusPaths() []string {
	return workspaceStatusPaths(WorkspaceRoot())
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
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core.
//
//	root := agentic.CoreRoot()
func CoreRoot() string {
	if root := core.Env("CORE_WORKSPACE"); root != "" {
		return root
	}
	return core.JoinPath(core.Env("DIR_HOME"), "Code", ".core")
}

func workspaceStatusPaths(wsRoot string) []string {
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))
	return append(old, deep...)
}

func workspaceRepoDir(wsDir string) string {
	return core.JoinPath(wsDir, "repo")
}

func workspaceMetaDir(wsDir string) string {
	return core.JoinPath(wsDir, ".meta")
}

func workspaceBlockedPath(wsDir string) string {
	return core.JoinPath(workspaceRepoDir(wsDir), "BLOCKED.md")
}

func workspaceAnswerPath(wsDir string) string {
	return core.JoinPath(workspaceRepoDir(wsDir), "ANSWER.md")
}

func workspaceLogFiles(wsDir string) []string {
	return core.PathGlob(core.JoinPath(workspaceMetaDir(wsDir), "agent-*.log"))
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
	if ref := s.gitOutput(ctx, repoDir, "symbolic-ref", "refs/remotes/origin/HEAD", "--short"); ref != "" {
		if core.HasPrefix(ref, "origin/") {
			return core.TrimPrefix(ref, "origin/")
		}
		return ref
	}
	for _, branch := range []string{"main", "master"} {
		if s.gitCmdOK(ctx, repoDir, "rev-parse", "--verify", branch) {
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
