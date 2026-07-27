// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- intValueOK ---

func TestPlatformValue_IntValueOK_Good_Int(t *testing.T) {
	got, ok := intValueOK(7)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 7, got)
}

func TestPlatformValue_IntValueOK_Good_Int64(t *testing.T) {
	got, ok := intValueOK(int64(9))
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 9, got)
}

func TestPlatformValue_IntValueOK_Good_Float64(t *testing.T) {
	got, ok := intValueOK(float64(3.9))
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 3, got)
}

func TestPlatformValue_IntValueOK_Good_StringNumber(t *testing.T) {
	got, ok := intValueOK("42")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 42, got)
}

func TestPlatformValue_IntValueOK_Good_StringZero(t *testing.T) {
	got, ok := intValueOK("0")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 0, got)
}

func TestPlatformValue_IntValueOK_Bad_NonNumberString(t *testing.T) {
	_, ok := intValueOK("abc")
	core.AssertFalse(t, ok)
}

func TestPlatformValue_IntValueOK_Bad_UnsupportedType(t *testing.T) {
	_, ok := intValueOK(true)
	core.AssertFalse(t, ok)
}

// --- intValue ---

func TestPlatformValue_IntValue_Good_AllNumericKinds(t *testing.T) {
	core.AssertEqual(t, 5, intValue(5))
	core.AssertEqual(t, 6, intValue(int64(6)))
	core.AssertEqual(t, 7, intValue(float64(7.8)))
	core.AssertEqual(t, 8, intValue("8"))
}

func TestPlatformValue_IntValue_Ugly_ZeroString(t *testing.T) {
	core.AssertEqual(t, 0, intValue("0"))
}

func TestPlatformValue_IntValue_Bad_NonNumberString(t *testing.T) {
	core.AssertEqual(t, 0, intValue("notanumber"))
}

func TestPlatformValue_IntValue_Bad_UnsupportedType(t *testing.T) {
	core.AssertEqual(t, 0, intValue([]string{"x"}))
}

// --- floatValue ---

func TestPlatformValue_FloatValue_Good_Float64(t *testing.T) {
	core.AssertEqual(t, 1.5, floatValue(float64(1.5)))
}

func TestPlatformValue_FloatValue_Good_Float32(t *testing.T) {
	core.AssertEqual(t, float64(float32(2.5)), floatValue(float32(2.5)))
}

func TestPlatformValue_FloatValue_Good_Int(t *testing.T) {
	core.AssertEqual(t, 3.0, floatValue(3))
}

func TestPlatformValue_FloatValue_Good_Int64(t *testing.T) {
	core.AssertEqual(t, 4.0, floatValue(int64(4)))
}

func TestPlatformValue_FloatValue_Good_String(t *testing.T) {
	core.AssertEqual(t, 5.25, floatValue("5.25"))
}

func TestPlatformValue_FloatValue_Ugly_EmptyString(t *testing.T) {
	core.AssertEqual(t, 0.0, floatValue(""))
}

func TestPlatformValue_FloatValue_Bad_InvalidString(t *testing.T) {
	core.AssertEqual(t, 0.0, floatValue("not a float"))
}

func TestPlatformValue_FloatValue_Bad_UnsupportedType(t *testing.T) {
	core.AssertEqual(t, 0.0, floatValue(true))
}

// --- boolMapValue ---

func TestPlatformValue_BoolMapValue_Good_TypedPassthrough(t *testing.T) {
	in := map[string]bool{"a": true, "b": false}
	got := boolMapValue(in)
	core.AssertTrue(t, got["a"])
	core.AssertFalse(t, got["b"])
}

func TestPlatformValue_BoolMapValue_Good_AnyMapMixedValues(t *testing.T) {
	got := boolMapValue(map[string]any{
		"flag_bool":   true,
		"flag_str":    "true",
		"flag_strno":  "false",
		"flag_int":    2,
		"flag_intneg": 0,
	})
	core.AssertTrue(t, got["flag_bool"])
	core.AssertTrue(t, got["flag_str"])
	core.AssertFalse(t, got["flag_strno"])
	core.AssertTrue(t, got["flag_int"])
	core.AssertFalse(t, got["flag_intneg"])
}

func TestPlatformValue_BoolMapValue_Good_JSONStringBoolMap(t *testing.T) {
	got := boolMapValue(`{"x":true,"y":false}`)
	core.AssertTrue(t, got["x"])
	core.AssertFalse(t, got["y"])
}

func TestPlatformValue_BoolMapValue_Good_JSONStringGenericMap(t *testing.T) {
	got := boolMapValue(`{"x":"true","y":0}`)
	core.AssertTrue(t, got["x"])
	core.AssertFalse(t, got["y"])
}

func TestPlatformValue_BoolMapValue_Ugly_EmptyString(t *testing.T) {
	core.AssertNil(t, boolMapValue(""))
}

func TestPlatformValue_BoolMapValue_Bad_UnsupportedType(t *testing.T) {
	core.AssertNil(t, boolMapValue(123))
}

// --- computeBudgetFromValue / computeBudgetFromMap ---

func TestPlatformValue_ComputeBudgetFromValue_Good_TypedPointer(t *testing.T) {
	in := &ComputeBudget{MaxDailyHours: 4}
	got := computeBudgetFromValue(in)
	core.RequireTrue(t, got != nil)
	core.AssertEqual(t, 4.0, got.MaxDailyHours)
}

func TestPlatformValue_ComputeBudgetFromValue_Ugly_NilTypedPointer(t *testing.T) {
	var in *ComputeBudget
	core.AssertNil(t, computeBudgetFromValue(in))
}

func TestPlatformValue_ComputeBudgetFromValue_Ugly_ZeroTypedPointer(t *testing.T) {
	in := &ComputeBudget{}
	core.AssertNil(t, computeBudgetFromValue(in))
}

func TestPlatformValue_ComputeBudgetFromValue_Good_TypedValue(t *testing.T) {
	got := computeBudgetFromValue(ComputeBudget{MaxWeeklyCostUSD: 100})
	core.RequireTrue(t, got != nil)
	core.AssertEqual(t, 100.0, got.MaxWeeklyCostUSD)
}

func TestPlatformValue_ComputeBudgetFromValue_Ugly_ZeroTypedValue(t *testing.T) {
	core.AssertNil(t, computeBudgetFromValue(ComputeBudget{}))
}

func TestPlatformValue_ComputeBudgetFromValue_Good_Map(t *testing.T) {
	got := computeBudgetFromValue(map[string]any{
		"max_daily_hours":     6.0,
		"max_weekly_cost_usd": 50.0,
		"quiet_start":         "22:00",
		"quiet_end":           "06:00",
		"prefer_models":       []any{"gemma"},
		"avoid_models":        []any{"gpt"},
	})
	core.RequireTrue(t, got != nil)
	core.AssertEqual(t, 6.0, got.MaxDailyHours)
	core.AssertEqual(t, 50.0, got.MaxWeeklyCostUSD)
	core.AssertEqual(t, "22:00", got.QuietStart)
	core.AssertEqual(t, "06:00", got.QuietEnd)
	core.AssertEqual(t, []string{"gemma"}, got.PreferModels)
	core.AssertEqual(t, []string{"gpt"}, got.AvoidModels)
}

func TestPlatformValue_ComputeBudgetFromValue_Good_MapStringString(t *testing.T) {
	got := computeBudgetFromValue(map[string]string{"max_daily_hours": "3"})
	core.RequireTrue(t, got != nil)
	core.AssertEqual(t, 3.0, got.MaxDailyHours)
}

func TestPlatformValue_ComputeBudgetFromValue_Good_JSONString(t *testing.T) {
	got := computeBudgetFromValue(`{"max_daily_hours":2}`)
	core.RequireTrue(t, got != nil)
	core.AssertEqual(t, 2.0, got.MaxDailyHours)
}

func TestPlatformValue_ComputeBudgetFromValue_Ugly_EmptyString(t *testing.T) {
	core.AssertNil(t, computeBudgetFromValue(""))
}

func TestPlatformValue_ComputeBudgetFromValue_Ugly_EmptyMap(t *testing.T) {
	core.AssertNil(t, computeBudgetFromValue(map[string]any{}))
}

func TestPlatformValue_ComputeBudgetFromValue_Ugly_ZeroValuesMap(t *testing.T) {
	core.AssertNil(t, computeBudgetFromValue(map[string]any{"max_daily_hours": 0.0}))
}

func TestPlatformValue_ComputeBudgetFromValue_Bad_UnsupportedType(t *testing.T) {
	core.AssertNil(t, computeBudgetFromValue(42))
}

// --- boolValueOK (auth.go) ---

func TestPlatformValue_BoolValueOK_Good_Bool(t *testing.T) {
	got, ok := boolValueOK(true)
	core.AssertTrue(t, ok)
	core.AssertTrue(t, got)
}

func TestPlatformValue_BoolValueOK_Good_StringTruthy(t *testing.T) {
	for _, in := range []string{"true", "1", "yes", "TRUE", " Yes "} {
		got, ok := boolValueOK(in)
		core.AssertTrue(t, ok, in)
		core.AssertTrue(t, got, in)
	}
}

func TestPlatformValue_BoolValueOK_Good_StringFalsy(t *testing.T) {
	for _, in := range []string{"false", "0", "no", "NO"} {
		got, ok := boolValueOK(in)
		core.AssertTrue(t, ok, in)
		core.AssertFalse(t, got, in)
	}
}

func TestPlatformValue_BoolValueOK_Good_IntKinds(t *testing.T) {
	got, ok := boolValueOK(1)
	core.AssertTrue(t, ok)
	core.AssertTrue(t, got)

	got, ok = boolValueOK(int64(0))
	core.AssertTrue(t, ok)
	core.AssertFalse(t, got)

	got, ok = boolValueOK(float64(2.0))
	core.AssertTrue(t, ok)
	core.AssertTrue(t, got)
}

func TestPlatformValue_BoolValueOK_Bad_UnknownString(t *testing.T) {
	_, ok := boolValueOK("maybe")
	core.AssertFalse(t, ok)
}

func TestPlatformValue_BoolValueOK_Bad_UnsupportedType(t *testing.T) {
	_, ok := boolValueOK([]int{1})
	core.AssertFalse(t, ok)
}
