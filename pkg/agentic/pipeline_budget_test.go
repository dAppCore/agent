// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineBudget_Good_RootHelp(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	output := captureStdout(t, func() {
		result := s.cmdPipelineBudget(core.NewOptions())
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "core-agent pipeline/budget/plan")
	assert.Contains(t, output, "core-agent pipeline/budget/log")
}

func TestPipelineBudget_Bad_PlanBlockedOnSibling(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineBudgetPlan(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "blocked-on-sibling")
}
