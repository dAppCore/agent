// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExampleSyncStatusInput() {
	input := SyncStatusInput{AgentID: "charon"}
	core.Println(input.AgentID)
	// Output: charon
}
