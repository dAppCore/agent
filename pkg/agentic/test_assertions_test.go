// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	core "dappco.re/go"
)

func assertRegexp(t *testing.T, pattern, value string, msg ...string) {
	t.Helper()

	matched, err := regexp.MatchString(pattern, value)
	core.RequireNoError(t, err)
	core.AssertTrue(t, matched, msg...)
}

func assertIsType(t *testing.T, want, got any, msg ...string) {
	t.Helper()
	core.AssertEqual(t, reflect.TypeOf(want), reflect.TypeOf(got), msg...)
}

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

func assertZero(t *testing.T, got any, msg ...string) {
	t.Helper()
	if got == nil {
		core.AssertNil(t, got, msg...)
		return
	}
	core.AssertEqual(t, reflect.Zero(reflect.TypeOf(got)).Interface(), got, msg...)
}

func assertNotZero(t *testing.T, got any, msg ...string) {
	t.Helper()
	if got == nil {
		t.Fatalf("assertNotZero requires a non-nil value")
	}
	core.AssertNotEqual(t, reflect.Zero(reflect.TypeOf(got)).Interface(), got, msg...)
}

func requireEventually(t *testing.T, condition func() bool, waitFor, tick time.Duration, msg ...string) {
	t.Helper()

	deadline := time.Now().Add(waitFor)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(tick)
	}

	if condition() {
		return
	}
	if len(msg) > 0 {
		t.Fatal(msg[0])
	}
	t.Fatalf("condition not satisfied within %s", waitFor)
}
