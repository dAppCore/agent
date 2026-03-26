// SPDX-License-Identifier: EUPL-1.2

package agentic_test

import (
	"context"

	core "dappco.re/go/core"

	"dappco.re/go/agent/pkg/agentic"
)

func ExampleProcessRegister() {
	c := core.New(
		core.WithService(agentic.ProcessRegister),
	)
	c.ServiceStartup(context.Background(), nil)

	core.Println(c.Process().Exists())
	// Output: true
}
