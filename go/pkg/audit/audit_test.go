// SPDX-License-Identifier: EUPL-1.2

package audit

import (
	"testing"

	core "dappco.re/go"
)

// TestAudit_Sanitise_Good — non-credential keys survive unchanged.
func TestAudit_Sanitise_Good(t *testing.T) {
	in := map[string]any{"profile": "default", "sandbox_id": "oc-1", "restarted": 2}
	out := Sanitise(in)
	if out["profile"] != "default" || out["sandbox_id"] != "oc-1" || out["restarted"] != 2 {
		t.Fatalf("benign keys dropped: %#v", out)
	}
}

// TestAudit_Sanitise_Bad — credential-shaped keys are dropped.
func TestAudit_Sanitise_Bad(t *testing.T) {
	in := map[string]any{
		"profile":           "x",
		"OPENCODE_PASSWORD": "hunter2",
		"api_token":         "sk-abc",
		"Authorization":     "Bearer y",
		"provider_secret":   "z",
		"bytes":             "raw",
		"private_key":       "pk",
	}
	out := Sanitise(in)
	if _, ok := out["profile"]; !ok {
		t.Fatal("benign key profile dropped")
	}
	for _, banned := range []string{"OPENCODE_PASSWORD", "api_token", "Authorization", "provider_secret", "bytes", "private_key"} {
		if _, ok := out[banned]; ok {
			t.Fatalf("credential-shaped key survived sanitise: %q", banned)
		}
	}
}

// TestAudit_Sanitise_Ugly — empty / all-credential maps collapse to nil.
func TestAudit_Sanitise_Ugly(t *testing.T) {
	if Sanitise(nil) != nil {
		t.Fatal("nil meta must sanitise to nil")
	}
	if Sanitise(map[string]any{}) != nil {
		t.Fatal("empty meta must sanitise to nil")
	}
	if out := Sanitise(map[string]any{"token": "x", "secret": "y"}); out != nil {
		t.Fatalf("all-credential map must sanitise to nil, got %#v", out)
	}
}

// TestAudit_FileSink_Good — Emit appends a JSONL record carrying the
// safe fields, stamps a timestamp, and sanitises Meta.
func TestAudit_FileSink_Good(t *testing.T) {
	fs := (&core.Fs{}).New("/")
	tempResult := fs.TempDir("core-audit-test")
	if !tempResult.OK {
		t.Fatalf("create temp dir: %v", tempResult.Value)
	}
	dir := tempResult.Value.(string)
	defer fs.DeleteAll(dir)
	path := core.JoinPath(dir, "audit.jsonl")

	sink := NewFileSink(fs, path)
	sink.Emit(Event{
		Event:      "opencode.sandbox.spawn",
		Outcome:    "ok",
		RequestID:  "req-1",
		SandboxID:  "oc-7f3a",
		PathPrefix: "/global",
		Meta:       map[string]any{"profile": "default", "secret": "leak"},
	})

	r := fs.Read(path)
	if !r.OK {
		t.Fatalf("read audit file: %v", r.Value)
	}
	body, _ := r.Value.(string)
	for _, want := range []string{`"event":"opencode.sandbox.spawn"`, `"outcome":"ok"`, `"sandbox_id":"oc-7f3a"`, `"path_prefix":"/global"`, `"profile":"default"`, `"at":`} {
		if !core.Contains(body, want) {
			t.Fatalf("audit record missing %q in:\n%s", want, body)
		}
	}
	if core.Contains(body, "secret") || core.Contains(body, "leak") {
		t.Fatalf("credential survived to disk:\n%s", body)
	}
}

// TestAudit_FileSink_Bad — a nil sink / empty path Emit is a safe no-op.
func TestAudit_FileSink_Bad(t *testing.T) {
	var s *FileSink
	s.Emit(Event{Event: "x"}) // nil receiver must not panic

	fs := (&core.Fs{}).New("/")
	NewFileSink(fs, "").Emit(Event{Event: "x"}) // empty path must not panic
}

// TestAudit_FileSink_Ugly — repeated Emit appends multiple lines.
func TestAudit_FileSink_Ugly(t *testing.T) {
	fs := (&core.Fs{}).New("/")
	tempResult := fs.TempDir("core-audit-test")
	if !tempResult.OK {
		t.Fatalf("create temp dir: %v", tempResult.Value)
	}
	dir := tempResult.Value.(string)
	defer fs.DeleteAll(dir)
	path := core.JoinPath(dir, "audit.jsonl")

	sink := NewFileSink(fs, path)
	sink.Emit(Event{Event: "a", Outcome: "ok"})
	sink.Emit(Event{Event: "b", Outcome: "denied"})

	body := fs.Read(path).Value.(string)
	lines := 0
	for _, ch := range body {
		if ch == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d:\n%s", lines, body)
	}
}
