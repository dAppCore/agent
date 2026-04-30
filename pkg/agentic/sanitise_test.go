// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestSanitise_SanitiseBranchSlug_Good_Basic(t *testing.T) {
	result := sanitiseBranchSlug("Fix broken tests", 40)
	core.AssertEqual(t, "fix-broken-tests", result)
	core.AssertNotContains(t, result, " ")
}

func TestSanitise_SanitiseBranchSlug_Bad_Empty(t *testing.T) {
	result := sanitiseBranchSlug("", 40)
	core.AssertEqual(t, "", result)
	core.AssertEmpty(t, result)
}

func TestSanitise_SanitiseBranchSlug_Ugly_Truncate(t *testing.T) {
	result := sanitiseBranchSlug("a very long description that exceeds the limit", 10)
	core.AssertTrue(t, len(result) <= 10)
	core.AssertNotEmpty(t, result)
}
