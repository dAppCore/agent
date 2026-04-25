// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func (s *PrepSubsystem) cmdPipelineBudgetPlan(_ core.Options) core.Result {
	core.Print(nil, "status: not yet implemented")
	core.Print(nil, "reason: blocked-on-sibling")
	return core.Result{
		Value: core.E("agentic.cmdPipelineBudgetPlan", "not yet implemented - blocked-on-sibling", nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdPipelineBudgetLog(_ core.Options) core.Result {
	core.Print(nil, "status: not yet implemented")
	core.Print(nil, "reason: blocked-on-sibling")
	return core.Result{
		Value: core.E("agentic.cmdPipelineBudgetLog", "not yet implemented - blocked-on-sibling", nil),
		OK:    false,
	}
}
