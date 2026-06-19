// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestQAAnalysis_qaAnalysisCompatible_Bad_DifferentCategory — findings with
// different categories are not compatible.
func TestQAAnalysis_qaAnalysisCompatible_Bad_DifferentCategory(t *testing.T) {
	core.AssertFalse(t, qaAnalysisCompatible(
		QAFinding{Category: "lint"},
		QAFinding{Category: "security"},
	))
}
