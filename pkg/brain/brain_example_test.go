// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go"
)

func ExampleNew() {
	sub := New(nil)
	core.Println(sub.Name())
	// Output: brain
}

func ExampleRegister_services() {
	c := core.New(core.WithService(Register))
	core.Println(c.Services())
	// Output is non-deterministic (slice order), so no Output comment
}
