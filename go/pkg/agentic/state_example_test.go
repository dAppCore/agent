// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "github.com/dappcore/go"
)

func Example_statePath() {
	core.Println(core.PathBase(statePath("ax-follow-up")))
	// Output: ax-follow-up.json
}
