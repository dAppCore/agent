// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExampleAgentMessage() {
	message := AgentMessage{FromAgent: "codex", ToAgent: "charon"}
	core.Println(message.ToAgent)
	// Output: charon
}
