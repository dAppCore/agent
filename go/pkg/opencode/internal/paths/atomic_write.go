// SPDX-Licence-Identifier: EUPL-1.2

// atomic_write.go — the minimal opencode-local slice of the desktop
// paths.AtomicWriteWithVersion write surface.
//
// opencode runs inside a sandbox and owns exactly one persisted file
// shape: opencode.json (the merged host-config). That write is
// unconditional (no version frontmatter, no optimistic-lock check)
// and is NOT an at-rest-encrypted secret. The desktop's full 21-file
// pkg/paths drags a lock + fstype + at-rest + audit-emit machinery
// opencode never exercises, and an audit dependency the sandbox must
// not carry. This file ports ONLY the symbols host_config.go uses,
// preserving the substantive guarantees the merge write relies on:
//
//   - tmp + fsync + rename atomic replacement (no half-written file
//     ever visible at the final path, power-failure-safe).
//   - 0o600 file mode verbatim (the merged file embeds pre-existing
//     user provider apiKey blocks; a wider mode leaks them to
//     cross-user read).
//   - per-call randomised tmp suffix so two concurrent in-process
//     writers to the same path cannot race on a fixed staging path.
//   - a fault-injection hook (SetWriteTmpOpenFaultForTest) for the
//     interrupted-write coverage in host_config_mode_test.go.
//
// The optimistic-lock fields (IfVersion / IfMtime / IfMatchHash /
// IfNotExist) are retained on WriteInput so the call shape stays
// source-compatible with the desktop surface, but opencode only ever
// submits the unconditional Body-only shape.

package paths

import (
	core "dappco.re/go"
)

// File mode for new + replaced files. 0o600 matches the
// ~/Lethean/conf/opencode/ discipline — the merged opencode.json may
// embed user apiKey blocks and must not be cross-user readable.
const writeFileMode core.FileMode = 0o600

// Write-path error codes. Reserved schema — pattern-matched by the
// interrupted-write coverage in host_config_mode_test.go.
const (
	CodeWriteInvalidPath = "paths.write.invalid_path"
	CodeWriteOpenFailed  = "paths.write.open_failed"
	CodeWriteFsync       = "paths.write.fsync_failed"
	CodeWriteRename      = "paths.write.rename_failed"
)

// WriteInput is the call payload for AtomicWriteWithVersion.
//
// opencode submits only the unconditional Body-only shape; the
// optimistic-lock fields are retained for source-compatibility with
// the desktop write surface but are unused by the host-config merge.
//
// Usage example:
//
//	r := paths.AtomicWriteWithVersion(fpath, paths.WriteInput{
//	    Body: composed,
//	})
type WriteInput struct {
	// Body is the new file content, written verbatim.
	Body []byte

	// Timeout caps the wait for a write slot. Reserved for
	// source-compatibility; unused by the unconditional write.
	Timeout core.Duration
}

// WriteOutput is the success-path payload (returned in Result.Value
// of an OK Result).
type WriteOutput struct {
	Mtime core.Time `json:"mtime"`
	Hash  string    `json:"hash"`
}

// AtomicWriteWithVersion performs the tmp + fsync + rename sequence,
// returning Ok(WriteOutput) on success.
//
// Usage example:
//
//	r := paths.AtomicWriteWithVersion(fpath, paths.WriteInput{
//	    Body: newBytes,
//	})
//	if !r.OK { return r }
//	out := r.Value.(paths.WriteOutput)
func AtomicWriteWithVersion(path string, input WriteInput) core.Result {
	if path == "" {
		return core.Fail(core.NewCode(CodeWriteInvalidPath,
			"AtomicWriteWithVersion requires a non-empty path"))
	}

	// Two-phase write: tmp + fsync + rename. Per-call randomised tmp
	// suffix removes the fixed-staging-path race between two
	// concurrent in-process writers to the same path.
	var tmp string
	if rs := core.RandomString(8); rs.OK {
		tmp = path + ".tmp." + rs.Value.(string)
	} else {
		return core.Fail(core.E(CodeWriteOpenFailed,
			"random suffix: "+rs.Error(), nil))
	}

	var openR core.Result
	if writeTmpOpenFaultForTest != nil {
		openR = writeTmpOpenFaultForTest(tmp)
	} else {
		openR = core.OpenFile(tmp,
			core.O_CREATE|core.O_WRONLY|core.O_TRUNC, writeFileMode)
	}
	if !openR.OK {
		return core.Fail(core.E(CodeWriteOpenFailed,
			"open tmp: "+openR.Error(), nil))
	}
	f, _ := openR.Value.(*core.OSFile)
	if f == nil {
		return core.Fail(core.NewCode(CodeWriteOpenFailed,
			"open tmp returned nil file"))
	}
	if _, err := f.Write(input.Body); err != nil {
		_ = f.Close()
		_ = core.Remove(tmp)
		return core.Fail(core.E(CodeWriteOpenFailed, "write tmp", err))
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = core.Remove(tmp)
		return core.Fail(core.E(CodeWriteFsync, "fsync tmp", err))
	}
	if err := f.Close(); err != nil {
		_ = core.Remove(tmp)
		return core.Fail(core.E(CodeWriteOpenFailed, "close tmp", err))
	}
	if r := core.Rename(tmp, path); !r.OK {
		_ = core.Remove(tmp)
		return core.Fail(core.E(CodeWriteRename, "rename: "+r.Error(), nil))
	}

	// Post-write stat for the success envelope.
	out := WriteOutput{Hash: core.SHA256Hex(input.Body)}
	if newStat := core.Lstat(path); newStat.OK {
		if info, _ := newStat.Value.(core.FsFileInfo); info != nil {
			out.Mtime = info.ModTime()
		}
	}
	return core.Ok(out)
}

// writeTmpOpenFaultForTest is a fault-injection hook used by the
// interrupted-write coverage in host_config_mode_test.go to force a
// deterministic tmp-stage open failure. When non-nil it overrides the
// OpenFile call that stages the tmp file. Production code MUST NOT
// touch this — pair every test setter with t.Cleanup that resets it
// to nil.
var writeTmpOpenFaultForTest func(tmp string) core.Result

// SetWriteTmpOpenFaultForTest installs a fault-injection callback that
// AtomicWriteWithVersion consults in place of the tmp-stage OpenFile.
// Pass nil to disable. Test-only.
//
// Usage example:
//
//	paths.SetWriteTmpOpenFaultForTest(func(tmp string) core.Result {
//	    return core.Fail(core.NewCode(paths.CodeWriteOpenFailed, "simulated"))
//	})
//	t.Cleanup(func() { paths.SetWriteTmpOpenFaultForTest(nil) })
func SetWriteTmpOpenFaultForTest(fn func(tmp string) core.Result) {
	writeTmpOpenFaultForTest = fn
}
