// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	core "dappco.re/go/core"
)

func Example_workspaceStatusPath() {
	path := workspaceStatusPath("/srv/workspace/go-io-123")
	core.Println(core.HasSuffix(path, "status.json"))
	// Output: true
}
