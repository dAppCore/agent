// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go"

func ExampleRegister() {
	c := core.New(core.WithService(Register))
	// Assert the service this factory actually registers. Counting the
	// registry instead only ever worked while core auto-registered a CLI
	// alongside it, which it no longer does.
	core.Println(c.Service("brain").OK)
	// Output: true
}
