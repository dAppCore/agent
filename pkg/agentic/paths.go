// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	core "dappco.re/go/core"
)

// fs provides unrestricted filesystem access (root "/" = no sandbox).
//
//	r := fs.Read("/etc/hostname")
//	if r.OK { fmt.Println(r.Value.(string)) }
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
func LocalFs() *core.Fs { return fs }

// WorkspaceRoot returns the root directory for agent workspaces.
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core/workspace.
//
//	wsDir := filepath.Join(agentic.WorkspaceRoot(), "go-io-1774149757")
func WorkspaceRoot() string {
	return filepath.Join(CoreRoot(), "workspace")
}

// CoreRoot returns the root directory for core ecosystem files.
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core.
func CoreRoot() string {
	if root := os.Getenv("CORE_WORKSPACE"); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Code", ".core")
}

// PlansRoot returns the root directory for agent plans.
func PlansRoot() string {
	return filepath.Join(CoreRoot(), "plans")
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
	h := strings.ToLower(hostname)
	if strings.Contains(h, "snider") || strings.Contains(h, "studio") || strings.Contains(h, "mac") {
		return "cladius"
	}
	return "charon"
}

// DefaultBranch detects the default branch of a repo (main, master, etc.).
func DefaultBranch(repoDir string) string {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = repoDir
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		if strings.HasPrefix(ref, "origin/") {
			return strings.TrimPrefix(ref, "origin/")
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
func GitHubOrg() string {
	if org := os.Getenv("GITHUB_ORG"); org != "" {
		return org
	}
	return "dAppCore"
}
