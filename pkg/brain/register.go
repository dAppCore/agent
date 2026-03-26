// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Returns the DirectSubsystem — WithService auto-registers it.
//
//	core.New(
//	    core.WithService(brain.Register),
//	)
func Register(c *core.Core) core.Result {
	brn := NewDirect()
	return core.Result{Value: brn, OK: true}
}
