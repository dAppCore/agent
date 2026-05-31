// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"
)

// --- ProviderList ---

// TestProviderList_EmptyID_Bad — ProviderList with empty id must return
// Fail without making any HTTP calls.
func TestProviderList_EmptyID_Bad(t *testing.T) {
	var s *Service
	r := s.ProviderList("")
	// On nil service, the first call (core.Trim) returns "", which
	// immediately returns Fail. The nil receiver is not dereferenced
	// before the guard.
	if r.OK {
		t.Fatal("expected Fail for empty id, got OK")
	}
}
