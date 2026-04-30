// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go"

func ExampleSendInput() {
	input := SendInput{To: "charon", Content: "deploy complete"}
	core.Println(input.To)
	// Output: charon
}

func ExampleDirectSubsystem_RegisterMessagingTools() {
	core.Println(brainExampleToolCount((&DirectSubsystem{}).RegisterMessagingTools) > 0)
	// Output: true
}
