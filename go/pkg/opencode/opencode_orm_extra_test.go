// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestImportHost_ListImports_NoStore — listing imports/providers without a
// migrated store fails cleanly rather than panicking.
func TestImportHost_ListImports_NoStore(t *testing.T) {
	svc := newTestService(t)
	core.AssertFalse(t, svc.ListImports().OK)
	core.AssertFalse(t, svc.ListImportedProviders().OK)
	core.AssertFalse(t, svc.Status().OK)
	core.AssertFalse(t, svc.Inspect("oc-1").OK)
	if _, tr := svc.targetFor("oc-1"); tr.OK {
		t.Fatalf("targetFor should fail without a running sandbox")
	}
}
