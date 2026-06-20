// SPDX-Licence-Identifier: EUPL-1.2

// Extra coverage for the generic profile value validator and the no-op
// audit-outcome hooks. validateProfileAnyValue is reached via SaveProfile
// for provider/mcp/agent option subtrees; the string + map arms are covered
// by the SaveProfile suite, but the []any recursion, the scalar arm, and the
// unsupported-type default were not. The emit* hooks are retained no-ops
// (the sandbox does not audit itself) and are exercised for completeness.

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestValidate_validateProfileAnyValue_AllArms — each value shape is walked:
// string (ok), nested map, nested slice, scalars, and an unsupported type.
func TestValidate_validateProfileAnyValue_AllArms(t *testing.T) {
	// string — ok.
	if err := validateProfileAnyValue("p", "hello"); err != nil {
		t.Fatalf("string value: unexpected error %v", err)
	}
	// nested map — recurses into children.
	if err := validateProfileAnyValue("p", map[string]any{"k": "v", "n": map[string]any{"d": "e"}}); err != nil {
		t.Fatalf("map value: unexpected error %v", err)
	}
	// nested slice — recurses by index.
	if err := validateProfileAnyValue("p", []any{"a", map[string]any{"b": "c"}}); err != nil {
		t.Fatalf("slice value: unexpected error %v", err)
	}
	// scalars — all ok.
	for _, v := range []any{nil, true, float64(1), int(2), int32(3), int64(4)} {
		if err := validateProfileAnyValue("p", v); err != nil {
			t.Fatalf("scalar %T: unexpected error %v", v, err)
		}
	}
}

// TestValidate_validateProfileAnyValue_StringRejected — an over-long string
// (or NUL byte) inside a map propagates the schema error out of the recursion.
func TestValidate_validateProfileAnyValue_StringRejected(t *testing.T) {
	bad := make([]byte, profileMaxStringLen+1)
	for i := range bad {
		bad[i] = 'x'
	}
	err := validateProfileAnyValue("p", map[string]any{"k": string(bad)})
	if err == nil {
		t.Fatal("expected schema error for over-long nested string")
	}
	if got := core.Fail(err).Code(); got != ProfileInvalidSchema {
		t.Fatalf("error code = %q; want %q", got, ProfileInvalidSchema)
	}
}

// TestValidate_validateProfileAnyValue_UnsupportedType — a value that cannot
// arrive from encoding/json (e.g. a channel) hits the default arm and is
// rejected as an unsupported value type.
func TestValidate_validateProfileAnyValue_UnsupportedType(t *testing.T) {
	err := validateProfileAnyValue("p", make(chan int))
	if err == nil {
		t.Fatal("expected schema error for unsupported value type")
	}
	if got := core.Fail(err).Code(); got != ProfileInvalidSchema {
		t.Fatalf("error code = %q; want %q", got, ProfileInvalidSchema)
	}
}

// TestValidate_AuditHooks_NoOp — the retained audit-outcome hooks are no-ops
// (the sandbox does not audit itself) and must not panic when called.
func TestValidate_AuditHooks_NoOp(t *testing.T) {
	emitSignatureVerified("sha256:abc", "key-1")
	emitSignatureRejected("sha256:abc", "key-1", "untrusted", core.Result{})
	(&Service{}).emitDenials("output", "install-123")
}
