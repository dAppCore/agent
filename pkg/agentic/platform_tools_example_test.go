// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleSyncStatusInput() {
	input := SyncStatusInput{AgentID: "charon"}
	core.Println(input.AgentID)
	// Output: charon
}
