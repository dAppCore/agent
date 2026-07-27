// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestService_Accessors — the service name/runtime/proxy accessors + the
// sandbox-change callback (incl. nil-receiver safety) behave as documented.
func TestService_Accessors(t *testing.T) {
	svc := newTestService(t)
	core.AssertEqual(t, "OpenCode", svc.ServiceName())
	core.AssertEqual(t, "docker", svc.runtime())
	core.AssertTrue(t, svc.ProxyGroup() != nil)

	fired := false
	svc.SetOnSandboxChange(func() { fired = true })
	svc.fireSandboxChange()
	core.AssertTrue(t, fired)

	// nil-receiver guards must not panic.
	var nilSvc *Service
	nilSvc.SetOnSandboxChange(func() {})
	nilSvc.fireSandboxChange()
}
