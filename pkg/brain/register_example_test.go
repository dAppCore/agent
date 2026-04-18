// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go/core"

func ExampleRegister() {
	c := core.New(core.WithService(Register))
	core.Println(len(c.Services()) > 1)
	// Output: true
}
