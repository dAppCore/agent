// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

func ExampleProcessRegister() {
	c := core.New()
	ProcessRegister(c)

	r := c.Action("process.run").Run(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "echo"},
		core.Option{Key: "args", Value: []string{"ok"}},
	))
	if r.OK {
		core.Println(core.Trim(r.Value.(string)))
	}

	// Output: ok
}

func ExampleOverrideService_OnStartup() {
	svc := &processOverrideService{handlers: &processActionHandlers{}, core: core.New()}
	core.Println(svc.OnStartup(context.Background()).OK)
	// Output: true
}
