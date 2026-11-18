// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"github.com/dappcore/agent/go/pkg/agentic"
	core "github.com/dappcore/go"
)

func Example_sharedWorkspaceStatusPath() {
	path := agentic.WorkspaceStatusPath("/srv/workspace/core/go-io/task-5")
	core.Println(core.HasSuffix(path, "status.json"))
	// Output: true
}
