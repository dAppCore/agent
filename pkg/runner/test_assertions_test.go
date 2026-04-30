// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"reflect"
	"testing"

	core "dappco.re/go"
)

func assertNotSame(t *testing.T, want, got any, msg ...string) {
	t.Helper()

	wantValue := reflect.ValueOf(want)
	gotValue := reflect.ValueOf(got)
	if !wantValue.IsValid() || !gotValue.IsValid() {
		t.Fatalf("assertNotSame requires non-nil values")
	}
	if wantValue.Kind() != reflect.Pointer || gotValue.Kind() != reflect.Pointer {
		t.Fatalf("assertNotSame requires pointer values")
	}

	core.AssertFalse(t, wantValue.Pointer() == gotValue.Pointer(), msg...)
}
