// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

// ProcessRegister is the service factory for go-process.
// Delegates to process.Register — named Actions (process.run, process.start,
// process.kill, process.list, process.get) are registered during OnStartup.
//
//	core.New(
//	    core.WithService(agentic.ProcessRegister),
//	)
func ProcessRegister(c *core.Core) core.Result {
	return process.Register(c)
}
