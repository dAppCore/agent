// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"
)

// --- ContainerName ---

// TestContainerName_ReturnsPrefixPlusID_Good — ContainerName must
// prepend the canonical "lthn-opencode-" prefix to the given id.
func TestContainerName_ReturnsPrefixPlusID_Good(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"oc-1735843891234", "lthn-opencode-oc-1735843891234"},
		{"oc-abc123", "lthn-opencode-oc-abc123"},
		{"sandbox-1", "lthn-opencode-sandbox-1"},
		{"", "lthn-opencode-"},
	}

	for _, tt := range tests {
		got := ContainerName(tt.id)
		if got != tt.want {
			t.Errorf("ContainerName(%q) = %q; want %q", tt.id, got, tt.want)
		}
	}
}

// TestContainerName_Deterministic_Good — same input always produces
// same output.
func TestContainerName_Deterministic_Good(t *testing.T) {
	for i := range 10 {
		if ContainerName("test-id") != "lthn-opencode-test-id" {
			t.Fatalf("ContainerName not deterministic on iteration %d", i)
		}
	}
}

// --- Status constants ---

// TestStatusConstants_Values_Ugly — verify the canonical status strings
// are set correctly.
func TestStatusConstants_Values_Ugly(t *testing.T) {
	if StatusRunning != "running" {
		t.Errorf("StatusRunning = %q; want %q", StatusRunning, "running")
	}
	if StatusStopped != "stopped" {
		t.Errorf("StatusStopped = %q; want %q", StatusStopped, "stopped")
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed = %q; want %q", StatusFailed, "failed")
	}
}

// --- Sandbox struct ---

// TestSandbox_DefaultZeroValue_Ugly — a zero-value Sandbox must have
// empty string fields, zero int port, and zero status.
func TestSandbox_DefaultZeroValue_Ugly(t *testing.T) {
	var sb Sandbox
	if sb.ID != "" {
		t.Errorf("zero Sandbox.ID = %q; want empty", sb.ID)
	}
	if sb.Image != "" {
		t.Errorf("zero Sandbox.Image = %q; want empty", sb.Image)
	}
	if sb.HostPort != 0 {
		t.Errorf("zero Sandbox.HostPort = %d; want 0", sb.HostPort)
	}
	if sb.Status != "" {
		t.Errorf("zero Sandbox.Status = %q; want empty", sb.Status)
	}
}

// TestSandbox_FieldAssignment_Good — verify all Sandbox fields can be
// set and read back.
func TestSandbox_FieldAssignment_Good(t *testing.T) {
	sb := Sandbox{
		ID:       "oc-7f3a2b1c",
		Image:    "lthn/dev:latest",
		HostPort: 49152,
		Status:   StatusRunning,
	}
	if sb.ID != "oc-7f3a2b1c" {
		t.Errorf("ID = %q; want oc-7f3a2b1c", sb.ID)
	}
	if sb.Image != "lthn/dev:latest" {
		t.Errorf("Image = %q; want lthn/dev:latest", sb.Image)
	}
	if sb.HostPort != 49152 {
		t.Errorf("HostPort = %d; want 49152", sb.HostPort)
	}
	if sb.Status != StatusRunning {
		t.Errorf("Status = %q; want %q", sb.Status, StatusRunning)
	}
}

// TestSandbox_AllStatusValues_Ugly — each status constant must be
// assignable and distinct.
func TestSandbox_AllStatusValues_Ugly(t *testing.T) {
	statuses := []string{StatusRunning, StatusStopped, StatusFailed}
	for i, s1 := range statuses {
		for j, s2 := range statuses {
			if i != j && s1 == s2 {
				t.Errorf("status constants must be distinct: %d==%d (%q)", i, j, s1)
			}
		}
	}
}
