// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	core "dappco.re/go"
)

// TestBrainActions_IntFromAny_Good — every numeric and string-numeric branch
// coerces to an int (a float truncates, a numeric string parses, whitespace is
// trimmed).
func TestBrainActions_IntFromAny_Good(t *testing.T) {
	core.AssertEqual(t, 5, actionIntFromAny(5))
	core.AssertEqual(t, 7, actionIntFromAny(int64(7)))
	core.AssertEqual(t, 3, actionIntFromAny(3.9))
	core.AssertEqual(t, 42, actionIntFromAny("42"))
	core.AssertEqual(t, 42, actionIntFromAny("  42  "))
}

// TestBrainActions_IntFromAny_Bad — an empty, unparseable, or unhandled-type
// value is zero, never a panic.
func TestBrainActions_IntFromAny_Bad(t *testing.T) {
	core.AssertEqual(t, 0, actionIntFromAny(""))
	core.AssertEqual(t, 0, actionIntFromAny("not-a-number"))
	core.AssertEqual(t, 0, actionIntFromAny(true))
	core.AssertEqual(t, 0, actionIntFromAny(nil))
}

// TestBrainActions_FloatFromAny_Good — float32/float64/int/int64 and numeric
// strings all coerce to a float64.
func TestBrainActions_FloatFromAny_Good(t *testing.T) {
	core.AssertEqual(t, 3.5, actionFloatFromAny(3.5))
	core.AssertEqual(t, 2.5, actionFloatFromAny(float32(2.5)))
	core.AssertEqual(t, 4.0, actionFloatFromAny(4))
	core.AssertEqual(t, 6.0, actionFloatFromAny(int64(6)))
	core.AssertEqual(t, 1.5, actionFloatFromAny("1.5"))
}

// TestBrainActions_FloatFromAny_Bad — empty, unparseable, and unhandled-type
// values are zero.
func TestBrainActions_FloatFromAny_Bad(t *testing.T) {
	core.AssertEqual(t, 0.0, actionFloatFromAny(""))
	core.AssertEqual(t, 0.0, actionFloatFromAny("nope"))
	core.AssertEqual(t, 0.0, actionFloatFromAny(true))
	core.AssertEqual(t, 0.0, actionFloatFromAny(nil))
}

// TestBrainActions_StringFromAny_Good — numeric and bool inputs stringify (a
// float renders as its integer form), and strings are trimmed.
func TestBrainActions_StringFromAny_Good(t *testing.T) {
	core.AssertEqual(t, "5", actionStringFromAny(5))
	core.AssertEqual(t, "7", actionStringFromAny(int64(7)))
	core.AssertEqual(t, "3", actionStringFromAny(3.0))
	core.AssertEqual(t, "true", actionStringFromAny(true))
	core.AssertEqual(t, "trimmed", actionStringFromAny("  trimmed  "))
}

// TestBrainActions_StringFromAny_Bad — an unhandled type is the empty string.
func TestBrainActions_StringFromAny_Bad(t *testing.T) {
	core.AssertEqual(t, "", actionStringFromAny(nil))
	core.AssertEqual(t, "", actionStringFromAny([]int{1}))
}

// TestBrainActions_StringSliceFromAny_String_Good — a JSON-array string and a
// comma-separated string both normalise to a trimmed, empty-free slice.
func TestBrainActions_StringSliceFromAny_String_Good(t *testing.T) {
	core.AssertEqual(t, []string{"a", "b"}, actionStringSliceFromAny(`["a","b"]`))
	core.AssertEqual(t, []string{"a", "b", "c"}, actionStringSliceFromAny("a, b , c"))
}

// TestBrainActions_StringSliceFromAny_Ugly — an empty/nil value is nil; a scalar
// non-string falls back to a single stringified element.
func TestBrainActions_StringSliceFromAny_Ugly(t *testing.T) {
	core.AssertEqual(t, []string(nil), actionStringSliceFromAny(""))
	core.AssertEqual(t, []string(nil), actionStringSliceFromAny(nil))
	core.AssertEqual(t, []string{"5"}, actionStringSliceFromAny(5))
}

// TestBrainActions_RecallFilterValue_Good — a RecallFilter passes through, and
// both map shapes populate the typed fields.
func TestBrainActions_RecallFilterValue_Good(t *testing.T) {
	passthrough := RecallFilter{Org: "core", MinConfidence: 0.9}
	core.AssertEqual(t, passthrough, recallFilterValue(passthrough))

	fromAny := recallFilterValue(map[string]any{
		"project": "p", "type": "decision", "agent_id": "a", "org": "o", "min_confidence": 0.5,
	})
	core.AssertEqual(t, "p", fromAny.Project)
	core.AssertEqual(t, "a", fromAny.AgentID)
	core.AssertEqual(t, "o", fromAny.Org)
	core.AssertEqual(t, 0.5, fromAny.MinConfidence)
	core.AssertEqual(t, "decision", fromAny.Type)

	fromStrings := recallFilterValue(map[string]string{"project": "p2", "type": "t2", "org": "o2"})
	core.AssertEqual(t, "p2", fromStrings.Project)
	core.AssertEqual(t, "o2", fromStrings.Org)
	core.AssertEqual(t, "t2", fromStrings.Type)
}

// TestBrainActions_RecallFilterValue_Ugly — a bare string or scalar becomes a
// Type filter; an empty/unstringifiable value is the zero filter.
func TestBrainActions_RecallFilterValue_Ugly(t *testing.T) {
	core.AssertEqual(t, "memory", recallFilterValue("memory").Type)
	core.AssertEqual(t, "9", recallFilterValue(9).Type)
	core.AssertEqual(t, RecallFilter{}, recallFilterValue(nil))
}

// TestBrainActions_OptionValue_Good — the first matching key wins; an absent key
// set yields nil.
func TestBrainActions_OptionValue_Good(t *testing.T) {
	opts := core.NewOptions(core.Option{Key: "b", Value: "second"})
	core.AssertEqual(t, "second", actionOptionValue(opts, "a", "b"))
	core.AssertEqual(t, nil, actionOptionValue(opts, "missing"))
}
