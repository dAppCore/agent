// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"path/filepath"
)

// WorkspaceRoot returns the root directory for agent workspaces.
// Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core/workspace.
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
