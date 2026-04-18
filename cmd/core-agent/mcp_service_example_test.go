// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/core"
	"dappco.re/go/mcp/pkg/mcp"
)

func Example_mcpRegister() {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(mcp.Register),
	)

	result := c.Service("mcp")

	core.Println(result.OK)
	// Output: true
}
