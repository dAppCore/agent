// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	iofs "io/fs"
	"slices"
	"strconv"

	core "dappco.re/go/core"
)

// fsEntry matches the fs.DirEntry methods used by workspace scanning.
// Avoids importing io/fs — core.Fs.List() returns []iofs.DirEntry internally.
//
//	entry, ok := item.(fsEntry)
//	if ok { core.Print(nil, "%s isDir=%v", entry.Name(), entry.IsDir()) }
type fsEntry interface {
	Name() string
	IsDir() bool
}

// r := fs.Read("/etc/hostname")
// if r.OK { core.Print(nil, "%s", r.Value.(string)) }
var fs = (&core.Fs{}).NewUnrestricted()

var workspaceRootOverride string

func setWorkspaceRootOverride(root string) {
	workspaceRootOverride = core.Trim(root)
}

// f := agentic.LocalFs()
// r := f.Read("/tmp/agent-status.json")
func LocalFs() *core.Fs { return fs }

// workspaceDir := core.JoinPath(agentic.WorkspaceRoot(), "core", "go-io", "task-42")
func WorkspaceRoot() string {
	if root := core.Trim(workspaceRootOverride); root != "" {
		return root
	}
	return core.JoinPath(CoreRoot(), "workspace")
}

// paths := agentic.WorkspaceStatusPaths()
func WorkspaceStatusPaths() []string {
	return workspaceStatusPaths(WorkspaceRoot())
}

// path := agentic.WorkspaceStatusPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceStatusPath(workspaceDir string) string {
	return core.JoinPath(workspaceDir, "status.json")
}

// name := agentic.WorkspaceName("/Users/snider/Code/.core/workspace/core/go-io/dev")
func WorkspaceName(workspaceDir string) string {
	root := WorkspaceRoot()
	name := core.TrimPrefix(workspaceDir, root)
	name = core.TrimPrefix(name, "/")
	if name == "" {
		return core.PathBase(workspaceDir)
	}
	return name
}

// root := agentic.CoreRoot()
func CoreRoot() string {
	if root := core.Env("CORE_WORKSPACE"); root != "" {
		return root
	}
	return core.JoinPath(HomeDir(), "Code", ".core")
}

// home := agentic.HomeDir()
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

		for _, name := range listDirNames(r) {
			child := core.JoinPath(dir, name)
			if fs.IsDir(child) {
				walk(child, depth+1)
			}
		}
	}

	walk(workspaceRoot, 0)
	slices.Sort(paths)
	return paths
}

// listDirNames extracts entry names from a core.Fs.List() Result.
// core.Fs.List() returns []iofs.DirEntry — type-assert directly.
//
//	r := fs.List("/path/to/dir")
//	names := listDirNames(r) // ["file.go", "subdir", "README.md"]
func listDirNames(r core.Result) []string {
	if !r.OK || r.Value == nil {
		return nil
	}
	entries, ok := r.Value.([]iofs.DirEntry)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// listDirEntries extracts fsEntry values from a core.Fs.List() Result.
// core.Fs.List() returns []iofs.DirEntry — type-assert directly.
//
//	r := fs.List("/path/to/dir")
//	for _, entry := range listDirEntries(r) { core.Print(nil, "%s", entry.Name()) }
func listDirEntries(r core.Result) []fsEntry {
	if !r.OK || r.Value == nil {
		return nil
	}
	entries, ok := r.Value.([]iofs.DirEntry)
	if !ok {
		return nil
	}
	result := make([]fsEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	return result
}

// repoDir := agentic.WorkspaceRepoDir("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceRepoDir(workspaceDir string) string {
	return core.JoinPath(workspaceDir, "repo")
}

func workspaceRepoDir(workspaceDir string) string {
	return WorkspaceRepoDir(workspaceDir)
}

// metaDir := agentic.WorkspaceMetaDir("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceMetaDir(workspaceDir string) string {
	return core.JoinPath(workspaceDir, ".meta")
}

func workspaceMetaDir(workspaceDir string) string {
	return WorkspaceMetaDir(workspaceDir)
}

// blocked := agentic.WorkspaceBlockedPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceBlockedPath(workspaceDir string) string {
	return core.JoinPath(WorkspaceRepoDir(workspaceDir), "BLOCKED.md")
}

func workspaceBlockedPath(workspaceDir string) string {
	return WorkspaceBlockedPath(workspaceDir)
}

// answer := agentic.WorkspaceAnswerPath("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceAnswerPath(workspaceDir string) string {
	return core.JoinPath(WorkspaceRepoDir(workspaceDir), "ANSWER.md")
}

func workspaceAnswerPath(workspaceDir string) string {
	return WorkspaceAnswerPath(workspaceDir)
}

// logs := agentic.WorkspaceLogFiles("/srv/.core/workspace/core/go-io/task-5")
func WorkspaceLogFiles(workspaceDir string) []string {
	return core.PathGlob(core.JoinPath(WorkspaceMetaDir(workspaceDir), "agent-*.log"))
}

func workspaceLogFiles(workspaceDir string) []string {
	return WorkspaceLogFiles(workspaceDir)
}

// plansDir := agentic.PlansRoot()
func PlansRoot() string {
	return core.JoinPath(CoreRoot(), "plans")
}

// name := agentic.AgentName() // "cladius" on Snider's Mac, "charon" elsewhere
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

// base := s.DefaultBranch("/srv/Code/core/go-io/repo")
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

// org := agentic.GitHubOrg() // "dAppCore"
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
