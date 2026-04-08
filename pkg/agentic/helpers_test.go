// SPDX-License-Identifier: EUPL-1.2

package agentic

import "testing"

// setTestWorkspace sets CORE_WORKSPACE and clears the package-level
// workspaceRootOverride so tests aren't poisoned by prior test runs
// that called setWorkspaceRootOverride via loadAgentConfig.
func setTestWorkspace(t *testing.T, root string) {
	t.Helper()
	t.Setenv("CORE_WORKSPACE", root)
	setWorkspaceRootOverride("")
	t.Cleanup(func() { setWorkspaceRootOverride("") })
}
