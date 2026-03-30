// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	iofs "io/fs"
	"sort"
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
//	workspaceDir := core.JoinPath(agentic.WorkspaceRoot(), "core", "go-io", "task-42")
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
func WorkspaceStatusPath(workspaceDir string) string {
	return core.JoinPath(workspaceDir, "status.json")
}

// WorkspaceName extracts the unique workspace name from a full path.
// Given /Users/snider/Code/.core/workspace/core/go-io/dev → core/go-io/dev
//
//	name := agentic.WorkspaceName("/Users/snider/Code/.core/workspace/core/go-io/dev")
func WorkspaceName(workspaceDir string) string {
	root := WorkspaceRoot()
	name := core.TrimPrefix(workspaceDir, root)
	name = core.TrimPrefix(name, "/")
	if name == "" {
		return core.PathBase(workspaceDir)
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

func workspaceStatusPaths(workspaceRoot string) []string {
	if workspaceRoot == "" {
		return nil
	}

	var paths []string
	seen := make(map[string]bool)

	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		r := fs.List(dir)
		if !r.OK {
			return
		}

		entries, ok := r.Value.([]iofs.DirEntry)
		if !ok {
			return
		}

		statusPath := core.JoinPath(dir, "status.json")
		if fs.IsFile(statusPath) {
			if depth == 1 || depth == 3 || (fs.IsDir(core.JoinPath(dir, "repo")) && fs.IsDir(core.JoinPath(dir, ".meta"))) {
				if !seen[statusPath] {
					seen[statusPath] = true
					paths = append(paths, statusPath)
				}
				return
			}
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			walk(core.JoinPath(dir, entry.Name()), depth+1)
		}
	}

	walk(workspaceRoot, 0)
	sort.Strings(paths)
	return paths
}

// WorkspaceRepoDir returns the checked-out repo directory for a workspace.
//
//	repoDir := agentic.WorkspaceRepoDir("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceRepoDir(workspaceDir string) string {
	return core.JoinPath(workspaceDir, "repo")
}

func workspaceRepoDir(workspaceDir string) string {
	return WorkspaceRepoDir(workspaceDir)
}

// WorkspaceMetaDir returns the metadata directory for a workspace.
//
//	metaDir := agentic.WorkspaceMetaDir("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceMetaDir(workspaceDir string) string {
	return core.JoinPath(workspaceDir, ".meta")
}

func workspaceMetaDir(workspaceDir string) string {
	return WorkspaceMetaDir(workspaceDir)
}

// WorkspaceBlockedPath returns the BLOCKED.md path for a workspace.
//
//	blocked := agentic.WorkspaceBlockedPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceBlockedPath(workspaceDir string) string {
	return core.JoinPath(WorkspaceRepoDir(workspaceDir), "BLOCKED.md")
}

func workspaceBlockedPath(workspaceDir string) string {
	return WorkspaceBlockedPath(workspaceDir)
}

// WorkspaceAnswerPath returns the ANSWER.md path for a workspace.
//
//	answer := agentic.WorkspaceAnswerPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceAnswerPath(workspaceDir string) string {
	return core.JoinPath(WorkspaceRepoDir(workspaceDir), "ANSWER.md")
}

func workspaceAnswerPath(workspaceDir string) string {
	return WorkspaceAnswerPath(workspaceDir)
}

// WorkspaceLogFiles returns captured agent log files for a workspace.
//
//	logs := agentic.WorkspaceLogFiles("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceLogFiles(workspaceDir string) []string {
	return core.PathGlob(core.JoinPath(WorkspaceMetaDir(workspaceDir), "agent-*.log"))
}

func workspaceLogFiles(workspaceDir string) []string {
	return WorkspaceLogFiles(workspaceDir)
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
