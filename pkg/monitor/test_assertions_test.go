// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"testing"
	"time"
)

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
