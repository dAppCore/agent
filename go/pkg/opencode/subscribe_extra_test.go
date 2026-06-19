// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestService_Subscribe — subscribe rejects empty ids + nil receivers;
// unsubscribe is a safe no-op for unknown ids.
func TestService_Subscribe(t *testing.T) {
	svc := newTestService(t)

	_, r := svc.Subscribe("")
	core.AssertFalse(t, r.OK)

	cancel, _ := svc.Subscribe("oc-1")
	cancel()
	svc.Unsubscribe("oc-1")
	svc.Unsubscribe("nope")

	var nilSvc *Service
	_, rn := nilSvc.Subscribe("x")
	core.AssertFalse(t, rn.OK)
	nilSvc.Unsubscribe("x")
}
