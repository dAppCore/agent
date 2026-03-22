// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"os/exec"
	"strconv"
	"unsafe"

	core "dappco.re/go/core"
)

// fs provides unrestricted filesystem access (root "/" = no sandbox).
//
//	r := fs.Read("/etc/hostname")
//	if r.OK { core.Print(nil, "%s", r.Value.(string)) }
var fs = newFs("/")

// newFs creates a core.Fs with the given root directory.
// Root "/" means unrestricted access (same as coreio.Local).
func newFs(root string) *core.Fs {
	type fsRoot struct{ root string }
	f := &core.Fs{}
	(*fsRoot)(unsafe.Pointer(f)).root = root
	return f
}

// LocalFs returns an unrestricted filesystem instance for use by other packages.
//
//	r := agentic.LocalFs().Read("/tmp/agent-status.json")
//	if r.OK { core.Print(nil, "%s", r.Value.(string)) }
func LocalFs() *core.Fs { return fs }

// WorkspaceRoot returns the root directory for agent workspaces.
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core/workspace.
//
//	wsDir := core.JoinPath(agentic.WorkspaceRoot(), "go-io-1774149757")
func WorkspaceRoot() string {
	return core.JoinPath(CoreRoot(), "workspace")
}

// CoreRoot returns the root directory for core ecosystem files.
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core.
//
//	root := agentic.CoreRoot()
func CoreRoot() string {
	if root := os.Getenv("CORE_WORKSPACE"); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return core.JoinPath(home, "Code", ".core")
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
	if name := os.Getenv("AGENT_NAME"); name != "" {
		return name
	}
	hostname, _ := os.Hostname()
	h := core.Lower(hostname)
	if core.Contains(h, "snider") || core.Contains(h, "studio") || core.Contains(h, "mac") {
		return "cladius"
	}
	return "charon"
}

// DefaultBranch detects the default branch of a repo (main, master, etc.).
//
//	base := agentic.DefaultBranch("./src")
func DefaultBranch(repoDir string) string {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = repoDir
	if out, err := cmd.Output(); err == nil {
		ref := core.Trim(string(out))
		if core.HasPrefix(ref, "origin/") {
			return core.TrimPrefix(ref, "origin/")
		}
		return ref
	}
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = repoDir
		if cmd.Run() == nil {
			return branch
		}
	}
	return "main"
}

// GitHubOrg returns the GitHub org for mirror operations.
//
//	org := agentic.GitHubOrg() // "dAppCore"
func GitHubOrg() string {
	if org := os.Getenv("GITHUB_ORG"); org != "" {
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
