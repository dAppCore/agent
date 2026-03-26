// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go/core"

func ExampleForgetInput() {
	input := ForgetInput{ID: "mem-123", Reason: "outdated"}
	core.Println(input.ID)
	// Output: mem-123
}
