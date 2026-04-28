// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	core "dappco.re/go"
)

func TestTools_ForgetInput_Good(t *testing.T) {
	input := ForgetInput{ID: "mem-123", Reason: "outdated"}
	core.AssertEqual(t, "mem-123", input.ID)
	core.AssertEqual(t, "outdated", input.Reason)
}

func TestTools_RememberInput_Good(t *testing.T) {
	input := RememberInput{Content: "Core uses Result", Type: "observation"}
	core.AssertEqual(t, "Core uses Result", input.Content)
	core.AssertEqual(t, "observation", input.Type)
}

func TestTools_RecallInput_Good(t *testing.T) {
	input := RecallInput{Query: "error handling", TopK: 10}
	core.AssertEqual(t, "error handling", input.Query)
	core.AssertEqual(t, 10, input.TopK)
}

func TestTools_Memory_Good(t *testing.T) {
	memory := Memory{DeletedAt: "2026-04-01T00:00:00Z"}

	core.AssertEqual(t, "2026-04-01T00:00:00Z", memory.DeletedAt)
	core.AssertContains(t, memory.DeletedAt, "2026")
}

func TestTools_RememberInput_Bad(t *testing.T) {
	var input RememberInput
	result := core.JSONUnmarshalString(`{"content":"Use core.Env for paths","type":42}`, &input)
	core.AssertFalse(t, result.OK)
}

func TestTools_RememberInput_Ugly(t *testing.T) {
	input := RememberInput{
		Content:    "Keep zero-value memory metadata intact",
		Type:       "observation",
		Tags:       nil,
		Confidence: 0,
		ExpiresIn:  0,
	}

	var output RememberInput
	roundTrip(t, input, &output)

	core.AssertEqual(t, input.Content, output.Content)
	core.AssertEqual(t, input.Type, output.Type)
	core.AssertNil(t, output.Tags)
	assertZero(t, output.Confidence)
	assertZero(t, output.ExpiresIn)
}
