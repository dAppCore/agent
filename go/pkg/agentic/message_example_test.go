// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleAgentMessage() {
	message := AgentMessage{FromAgent: "codex", ToAgent: "charon"}
	core.Println(message.ToAgent)
	// Output: charon
}
