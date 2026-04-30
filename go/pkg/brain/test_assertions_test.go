// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"reflect"
	"testing"
	"time"

	core "dappco.re/go"
)

func assertZero(t *testing.T, got any, msg ...string) {
	t.Helper()
	if got == nil {
		core.AssertNil(t, got, msg...)
		return
	}
	core.AssertEqual(t, reflect.Zero(reflect.TypeOf(got)).Interface(), got, msg...)
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
