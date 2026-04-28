// SPDX-License-Identifier: EUPL-1.2

package agentic_test

import (
	"context"

	core "dappco.re/go"

	"dappco.re/go/agent/pkg/agentic"
)

func ExampleRegister_fullService() {
	c := core.New(
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
	)
	c.ServiceStartup(context.Background(), nil)

	// All agentic Actions are now registered
	core.Println(c.Action("agentic.dispatch").Exists())
	core.Println(c.Action("agentic.status").Exists())
	c.ServiceShutdown(context.Background())
	// Output:
	// true
	// true
}

func ExampleProcessRegister() {
	c := core.New(
		core.WithService(agentic.ProcessRegister),
	)
	c.ServiceStartup(context.Background(), nil)

	core.Println(c.Process().Exists())
	// Output: true
}
