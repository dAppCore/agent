// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

func Example_sharedWorkspaceStatusPath() {
	path := agentic.WorkspaceStatusPath("/srv/workspace/core/go-io/task-5")
	core.Println(core.HasSuffix(path, "status.json"))
	// Output: true
}
