// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go/core"

func ExampleSendInput() {
	input := SendInput{To: "charon", Content: "deploy complete"}
	core.Println(input.To)
	// Output: charon
}
