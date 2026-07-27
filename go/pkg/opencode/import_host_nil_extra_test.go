// SPDX-Licence-Identifier: EUPL-1.2

// Nil-receiver guard coverage for the import read surface. ListImports /
// ListImportedProviders are called from the Wails + control layers, and a
// nil *Service must fail closed rather than nil-deref into the ORM. The
// success paths are covered by import_store_extra_test.go; these pin the
// s == nil short-circuit each method opens with.

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestImportHost_ListImports_NilReceiver_Bad — a nil *Service.ListImports
// returns the "service is nil" failure, never panicking.
func TestImportHost_ListImports_NilReceiver_Bad(t *testing.T) {
	var s *Service
	r := s.ListImports()
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "service is nil")
}

// TestImportHost_ListImportedProviders_NilReceiver_Bad — a nil
// *Service.ListImportedProviders fails closed identically.
func TestImportHost_ListImportedProviders_NilReceiver_Bad(t *testing.T) {
	var s *Service
	r := s.ListImportedProviders()
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "service is nil")
}
