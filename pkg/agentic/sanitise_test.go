// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitise_SanitiseBranchSlug_Good_Basic(t *testing.T) {
	assert.Equal(t, "fix-broken-tests", sanitiseBranchSlug("Fix broken tests", 40))
}

func TestSanitise_SanitiseBranchSlug_Bad_Empty(t *testing.T) {
	assert.Equal(t, "", sanitiseBranchSlug("", 40))
}

func TestSanitise_SanitiseBranchSlug_Ugly_Truncate(t *testing.T) {
	result := sanitiseBranchSlug("a very long description that exceeds the limit", 10)
	assert.True(t, len(result) <= 10)
}
