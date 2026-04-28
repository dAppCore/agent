// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	core "dappco.re/go"
)

func ExampleRegister_actions() {
	c := core.New(core.WithService(Register))
	c.ServiceStartup(context.Background(), nil)

	core.Println(c.Action("brain.list").Exists())
	core.Println(c.Action("message.send").Exists())
	// Output:
	// true
	// true
}
