// SPDX-License-Identifier: EUPL-1.2

package agentic_test

import (
	"context"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"

	"dappco.re/go/agent/pkg/agentic"
)

func ExampleRegister() {
	c := core.New(
		core.WithService(process.Register),
		core.WithService(agentic.Register),
	)
	c.ServiceStartup(context.Background(), nil)

	// All actions registered during OnStartup
	core.Println(c.Action("agentic.dispatch").Exists())
	core.Println(c.Action("agentic.status").Exists())
	core.Println(c.Action("agentic.qa").Exists())
	// Output:
	// true
	// true
	// true
}

func ExampleRegister_actions() {
	c := core.New(
		core.WithService(process.Register),
		core.WithService(agentic.Register),
	)
	c.ServiceStartup(context.Background(), nil)

	actions := c.Actions()
	core.Println(len(actions) > 0)
	// Output: true
}

func ExampleRegister_task() {
	c := core.New(
		core.WithService(process.Register),
		core.WithService(agentic.Register),
	)
	c.ServiceStartup(context.Background(), nil)

	// Completion pipeline registered as a Task
	t := c.Task("agent.completion")
	core.Println(t.Description)
	// Output: QA → PR → Verify → Merge
}
