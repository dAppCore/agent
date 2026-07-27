// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"net/http"
	"testing"

	core "dappco.re/go"
)

// TestOpencode_RegisterAndApplyAuth — Register builds + registers the service;
// applyAuth injects the persisted Basic-auth header onto a request.
func TestOpencode_RegisterAndApplyAuth(t *testing.T) {
	core.AssertTrue(t, Register(core.New()).OK)

	svc := newTestService(t)
	r, _ := http.NewRequest("GET", "http://127.0.0.1/x", nil)
	svc.applyAuth(r)
	core.AssertNotEmpty(t, r.Header.Get("Authorization"))
}
