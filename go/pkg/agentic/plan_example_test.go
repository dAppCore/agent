// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "github.com/dappcore/go"
)

func Example_planPath() {
	path := planPath("/tmp/plans", "my-plan")
	core.Println(core.HasSuffix(path, "my-plan.json"))
	// Output: true
}
