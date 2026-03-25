// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

// ProcessRegister is the service factory for the process management service.
// Wraps core/process for the v0.3.3→v0.4 factory pattern.
//
//	core.New(
//	    core.WithService(agentic.ProcessRegister),
//	)
func ProcessRegister(c *core.Core) core.Result {
	svc, err := process.NewService(process.Options{})(c)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	if procSvc, ok := svc.(*process.Service); ok {
		_ = process.SetDefault(procSvc)
	}
	return core.Result{Value: svc, OK: true}
}
