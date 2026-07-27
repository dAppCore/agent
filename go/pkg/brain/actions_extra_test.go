// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	core "dappco.re/go"
)

// TestBrainActions_ValueExtractors_Good — the float + string-slice option
// extractors pull typed values out of options.
func TestBrainActions_ValueExtractors_Good(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "f", Value: 1.5},
		core.Option{Key: "s", Value: []string{"a", "b"}},
	)
	core.AssertEqual(t, 1.5, actionFloatValue(opts, "f"))
	core.AssertEqual(t, []string{"a", "b"}, actionStringSliceValue(opts, "s"))
}

// TestBrainActions_StringSliceFromAny_Good — []string and []any inputs both
// normalise to a trimmed, empty-free slice.
func TestBrainActions_StringSliceFromAny_Good(t *testing.T) {
	core.AssertEqual(t, []string{"a", "b"}, actionStringSliceFromAny([]string{"a", " b ", ""}))
	core.AssertEqual(t, []string{"x", "y"}, actionStringSliceFromAny([]any{"x", "", "y"}))
}

// TestBrainActions_cleanActionStrings_Good — trims values and drops empties.
func TestBrainActions_cleanActionStrings_Good(t *testing.T) {
	core.AssertEqual(t, []string{"a", "b"}, cleanActionStrings([]string{" a ", "", "b", "  "}))
}

// TestBrainActions_MoreExtractors_Good — string/int option extractors + the
// any->string converter.
func TestBrainActions_MoreExtractors_Good(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "s", Value: "hello"},
		core.Option{Key: "n", Value: 7},
	)
	core.AssertEqual(t, "hello", actionStringValue(opts, "s"))
	core.AssertEqual(t, 7, actionIntValue(opts, "n"))
	core.AssertEqual(t, "x", actionStringFromAny("x"))
}
