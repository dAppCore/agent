// SPDX-Licence-Identifier: EUPL-1.2

package paths

import (
	"testing"

	core "dappco.re/go"
)

// --- AtomicWriteWithVersion ---

// TestAtomicWriteWithVersion_WritesContentAndReturnsOutput_Good —
// a clean write to a temp file must succeed, flush the body to disk,
// and return a WriteOutput with matching hash + non-zero mtime.
func TestAtomicWriteWithVersion_WritesContentAndReturnsOutput_Good(t *testing.T) {
	dir := t.TempDir()
	fpath := core.PathJoin(dir, "test.json")
	body := []byte(`{"provider":{"lthn":{}}}`)

	r := AtomicWriteWithVersion(fpath, WriteInput{Body: body})
	if !r.OK {
		t.Fatalf("AtomicWriteWithVersion failed: %v", r.Error())
	}
	out, ok := r.Value.(WriteOutput)
	if !ok {
		t.Fatalf("result is %T; want WriteOutput", r.Value)
	}
	if out.Hash != core.SHA256Hex(body) {
		t.Fatalf("hash = %q; want %q", out.Hash, core.SHA256Hex(body))
	}
	if out.Mtime.IsZero() {
		t.Fatal("mtime is zero; want non-zero")
	}

	// Verify file content on disk.
	readR := core.ReadFile(fpath)
	if !readR.OK {
		t.Fatalf("read back failed: %v", readR.Error())
	}
	if got := string(readR.Value.([]byte)); got != string(body) {
		t.Fatalf("file content = %q; want %q", got, string(body))
	}
}

// TestAtomicWriteWithVersion_EmptyPath_Bad — empty path must return
// CodeWriteInvalidPath without touching the filesystem.
func TestAtomicWriteWithVersion_EmptyPath_Bad(t *testing.T) {
	r := AtomicWriteWithVersion("", WriteInput{Body: []byte("x")})
	if r.OK {
		t.Fatal("expected Fail for empty path, got OK")
	}
	if r.Code() != CodeWriteInvalidPath {
		t.Fatalf("error code = %q; want %q", r.Code(), CodeWriteInvalidPath)
	}
}

// TestAtomicWriteWithVersion_ErrorCodes_Ugly — verify each error-code
// constant has the expected pattern prefix.
func TestAtomicWriteWithVersion_ErrorCodes_Ugly(t *testing.T) {
	codes := []string{CodeWriteInvalidPath, CodeWriteOpenFailed, CodeWriteFsync, CodeWriteRename}
	for _, code := range codes {
		if !core.HasPrefix(code, "paths.write.") {
			t.Fatalf("error code %q missing prefix 'paths.write.'", code)
		}
	}
}

// TestAtomicWriteWithVersion_BinaryBody_Good — binary body must survive
// the write/hash round-trip unchanged.
func TestAtomicWriteWithVersion_BinaryBody_Good(t *testing.T) {
	dir := t.TempDir()
	fpath := core.PathJoin(dir, "binary.bin")
	body := []byte{0x00, 0xFF, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E, 0x6F}

	r := AtomicWriteWithVersion(fpath, WriteInput{Body: body})
	if !r.OK {
		t.Fatalf("AtomicWriteWithVersion failed: %v", r.Error())
	}
	out, _ := r.Value.(WriteOutput)
	if out.Hash != core.SHA256Hex(body) {
		t.Fatalf("hash mismatch for binary body")
	}

	readR := core.ReadFile(fpath)
	if !readR.OK {
		t.Fatalf("read back failed: %v", readR.Error())
	}
	got := readR.Value.([]byte)
	if len(got) != len(body) {
		t.Fatalf("read back len = %d; want %d", len(got), len(body))
	}
	for i, b := range body {
		if got[i] != b {
			t.Fatalf("byte at offset %d = 0x%02X; want 0x%02X", i, got[i], b)
		}
	}
}

// TestAtomicWriteWithVersion_LargeBody_Good — a moderately large body
// must not hit any internal size limits.
func TestAtomicWriteWithVersion_LargeBody_Good(t *testing.T) {
	dir := t.TempDir()
	fpath := core.PathJoin(dir, "large.json")
	body := make([]byte, 128*1024) // 128 KiB
	for i := range body {
		body[i] = byte(i % 256)
	}

	r := AtomicWriteWithVersion(fpath, WriteInput{Body: body})
	if !r.OK {
		t.Fatalf("AtomicWriteWithVersion failed on 128 KiB: %v", r.Error())
	}
	out, _ := r.Value.(WriteOutput)
	if out.Hash != core.SHA256Hex(body) {
		t.Fatalf("hash mismatch for large body")
	}
}

// TestAtomicWriteWithVersion_OverwriteExisting_Good — writing to a path
// that already has content must replace it atomically.
func TestAtomicWriteWithVersion_OverwriteExisting_Good(t *testing.T) {
	dir := t.TempDir()
	fpath := core.PathJoin(dir, "overwrite.json")

	// Seed initial content.
	oldBody := []byte("old content")
	r1 := AtomicWriteWithVersion(fpath, WriteInput{Body: oldBody})
	if !r1.OK {
		t.Fatalf("first write failed: %v", r1.Error())
	}

	// Overwrite.
	newBody := []byte("new content")
	r2 := AtomicWriteWithVersion(fpath, WriteInput{Body: newBody})
	if !r2.OK {
		t.Fatalf("second write failed: %v", r2.Error())
	}

	readR := core.ReadFile(fpath)
	if !readR.OK {
		t.Fatalf("read back failed: %v", readR.Error())
	}
	if got := string(readR.Value.([]byte)); got != string(newBody) {
		t.Fatalf("file content = %q; want %q", got, string(newBody))
	}
}

// TestAtomicWriteWithVersion_NonExistentDir_Bad — write to a path where
// the parent dir doesn't exist must fail with a descriptive error.
func TestAtomicWriteWithVersion_NonExistentDir_Bad(t *testing.T) {
	dir := t.TempDir()
	fpath := core.PathJoin(dir, "nonexistent", "test.json")

	r := AtomicWriteWithVersion(fpath, WriteInput{Body: []byte("x")})
	if r.OK {
		t.Fatal("expected Fail for non-existent dir, got OK")
	}
	// The error will contain something about the write failing.
	msg := r.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
}

// --- SetWriteTmpOpenFaultForTest ---

// TestSetWriteTmpOpenFaultForTest_InjectsFault_Bad — installing a fault
// hook must cause AtomicWriteWithVersion to fail without modifying the
// target path.
func TestSetWriteTmpOpenFaultForTest_InjectsFault_Bad(t *testing.T) {
	SetWriteTmpOpenFaultForTest(func(tmp string) core.Result {
		return core.Fail(core.NewCode(CodeWriteOpenFailed, "simulated fault"))
	})
	t.Cleanup(func() { SetWriteTmpOpenFaultForTest(nil) })

	dir := t.TempDir()
	fpath := core.PathJoin(dir, "should-not-exist.json")

	r := AtomicWriteWithVersion(fpath, WriteInput{Body: []byte("x")})
	if r.OK {
		t.Fatal("expected Fail under fault injection, got OK")
	}
	msg := r.Error()
	if !core.Contains(msg, CodeWriteOpenFailed) {
		t.Fatalf("error = %q; want containing %q", msg, CodeWriteOpenFailed)
	}
	if !core.Contains(msg, "simulated fault") {
		t.Fatalf("error = %q; want containing 'simulated fault'", msg)
	}

	// Target file must not exist.
	if stat := core.Stat(fpath); stat.OK {
		t.Fatal("target file was created despite fault injection")
	}
}

// TestSetWriteTmpOpenFaultForTest_ResetWorks_Good — after clearing the
// fault hook, writes must succeed again.
func TestSetWriteTmpOpenFaultForTest_ResetWorks_Good(t *testing.T) {
	SetWriteTmpOpenFaultForTest(func(tmp string) core.Result {
		return core.Fail(core.NewCode(CodeWriteOpenFailed, "simulated"))
	})
	// Confirm fault active.
	dir := t.TempDir()
	fpath := core.PathJoin(dir, "fail.json")
	r := AtomicWriteWithVersion(fpath, WriteInput{Body: []byte("x")})
	if r.OK {
		t.Fatal("expected failure with fault hook")
	}

	// Clear and verify normal operation.
	SetWriteTmpOpenFaultForTest(nil)
	r = AtomicWriteWithVersion(fpath, WriteInput{Body: []byte("ok")})
	if !r.OK {
		t.Fatalf("expected OK after clearing fault: %v", r.Error())
	}
}

// TestSetWriteTmpOpenFaultForTest_NilHook_Good — passing nil must clear
// any previously installed hook.
func TestSetWriteTmpOpenFaultForTest_NilHook_Good(t *testing.T) {
	// Install a hook.
	SetWriteTmpOpenFaultForTest(func(tmp string) core.Result {
		return core.Fail(core.NewCode(CodeWriteOpenFailed, "fault"))
	})
	// Clear it.
	SetWriteTmpOpenFaultForTest(nil)

	dir := t.TempDir()
	fpath := core.PathJoin(dir, "normal.json")
	r := AtomicWriteWithVersion(fpath, WriteInput{Body: []byte("normal")})
	if !r.OK {
		t.Fatalf("expected OK after nil hook: %v", r.Error())
	}
}

// --- WriteInput / WriteOutput ---

// TestWriteInput_DefaultZeroValue_Ugly — zero WriteInput must have empty
// Body and zero Timeout.
func TestWriteInput_DefaultZeroValue_Ugly(t *testing.T) {
	var wi WriteInput
	if wi.Body != nil {
		t.Fatalf("zero WriteInput.Body = %v; want nil", wi.Body)
	}
	if wi.Timeout != 0 {
		t.Fatalf("zero WriteInput.Timeout = %v; want 0", wi.Timeout)
	}
}

// TestWriteOutput_Fields_Ugly — verify WriteOutput field assignment.
func TestWriteOutput_Fields_Ugly(t *testing.T) {
	out := WriteOutput{
		Mtime: core.Now(),
		Hash:  "abc123",
	}
	if out.Mtime.IsZero() {
		t.Fatal("Mtime must not be zero after assignment")
	}
	if out.Hash != "abc123" {
		t.Fatalf("Hash = %q; want abc123", out.Hash)
	}
}
