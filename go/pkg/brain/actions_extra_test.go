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
