// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"fmt"

	core "dappco.re/go/core"
)

func Example_statePath() {
	fmt.Println(core.PathBase(statePath("ax-follow-up")))
	// Output: ax-follow-up.json
}
