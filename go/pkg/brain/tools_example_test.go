// SPDX-License-Identifier: EUPL-1.2

package brain

import core "github.com/dappcore/go"

func ExampleForgetInput() {
	input := ForgetInput{ID: "mem-123", Reason: "outdated"}
	core.Println(input.ID)
	// Output: mem-123
}
