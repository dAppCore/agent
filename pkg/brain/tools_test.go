// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTools_ForgetInput_Good(t *testing.T) {
	input := ForgetInput{ID: "mem-123", Reason: "outdated"}
	assert.Equal(t, "mem-123", input.ID)
	assert.Equal(t, "outdated", input.Reason)
}

func TestTools_RememberInput_Good(t *testing.T) {
	input := RememberInput{Content: "Core uses Result", Type: "observation"}
	assert.Equal(t, "observation", input.Type)
}

func TestTools_RecallInput_Good(t *testing.T) {
	input := RecallInput{Query: "error handling", TopK: 10}
	assert.Equal(t, 10, input.TopK)
}

func TestTools_Memory_Good(t *testing.T) {
	memory := Memory{DeletedAt: "2026-04-01T00:00:00Z"}

	assert.Equal(t, "2026-04-01T00:00:00Z", memory.DeletedAt)
}
