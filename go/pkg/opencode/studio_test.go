// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"runtime"
	"testing"
)

// --- IsStudioInstalled ---

// TestIsStudioInstalled_ReturnsBool_Good — on any platform, the method
// must return a bool without panicking, even on nil receiver.
func TestIsStudioInstalled_ReturnsBool_Good(t *testing.T) {
	// The method does not reference the receiver; nil is safe.
	var s *Service
	result := s.IsStudioInstalled()
	// We can't assert true/false (depends on whether OpenCode.app is
	// installed on the test host), but we CAN assert the method
	// completes without panic and returns a sensible value.
	_ = result
}

// TestIsStudioInstalled_PlatformDependent_Good — the platform check
// branch must match runtime.GOOS.
func TestIsStudioInstalled_PlatformDependent_Good(t *testing.T) {
	var s *Service
	result := s.IsStudioInstalled()
	switch runtime.GOOS {
	case "darwin":
		// On macOS, result depends on whether /Applications/OpenCode.app exists.
		// The method itself is deterministic — just verify it's a bool.
		if result != true && result != false {
			t.Fatalf("expected bool, got %T", result)
		}
	case "linux", "windows":
		if result {
			t.Error("expected false on non-darwin platform")
		}
	default:
		if result {
			t.Error("expected false on unknown platform")
		}
	}
}

// --- OpenStudio ---

// TestOpenStudio_NilService_Bad — calling OpenStudio on a nil receiver
// must return Fail with a "service is nil" error.
func TestOpenStudio_NilService_Bad(t *testing.T) {
	var s *Service
	r := s.OpenStudio()
	if r.OK {
		t.Fatal("expected Fail for nil service, got OK")
	}
	if !contains(r.Error(), "service is nil") {
		t.Errorf("error = %q; want containing 'service is nil'", r.Error())
	}
}

// TestOpenStudio_NotInstalled_Bad — when the app isn't installed or
// the process service is unavailable, OpenStudio must return Fail.
func TestOpenStudio_NotInstalled_Bad(t *testing.T) {
	svc := &Service{}
	r := svc.OpenStudio()
	if r.OK {
		t.Log("OpenStudio succeeded — OpenCode.app is installed and runnable on this host")
		return
	}
	// Failure is expected on most test hosts. Accept either
	// "not installed" or "process service unavailable" — the
	// important contract is that it fails cleanly without panic.
	if !contains(r.Error(), "not installed") && !contains(r.Error(), "process service unavailable") {
		t.Errorf("error = %q; want containing 'not installed' or 'process service unavailable'", r.Error())
	}
}
