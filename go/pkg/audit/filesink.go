// SPDX-License-Identifier: EUPL-1.2

package audit

import (
	core "dappco.re/go"
)

// FileSink appends one JSON object per line (JSONL) to a file through
// c.Fs(), so audit writes stay sandbox-aware. Emit is concurrency-safe
// via an internal mutex — the hub's HTTP handlers call it from many
// goroutines.
//
// Usage example:
//
//	sink := audit.NewFileSink(c.Fs(), "/var/lib/core-agent/audit.jsonl")
//	sink.Emit(audit.Event{Event: "opencode.sandbox.spawn", Outcome: "ok"})
type FileSink struct {
	fs   *core.Fs
	path string
	mu   core.Mutex
}

var _ Sink = (*FileSink)(nil)

// NewFileSink constructs a JSONL file sink rooted at path. The parent
// directory is created lazily on the first Emit.
//
// Usage example:
//
//	sink := audit.NewFileSink(c.Fs(), audit.DefaultPath())
func NewFileSink(fs *core.Fs, path string) *FileSink {
	return &FileSink{fs: fs, path: path}
}

// Emit appends ev as one JSONL record. Meta is sanitised before the
// record is encoded so no credential-shaped key reaches disk. A zero At
// is stamped with the current time in RFC3339Nano. Failures are logged
// and swallowed — a broken audit file must not crash a spawn/stop, but
// the failure is surfaced in the process log so the operator notices a
// blind edge.
//
// Usage example:
//
//	sink.Emit(audit.Event{Event: "opencode.upgrade", Outcome: "ok"})
func (s *FileSink) Emit(ev Event) {
	if s == nil || s.fs == nil || core.Trim(s.path) == "" {
		return
	}
	if ev.At == "" {
		ev.At = core.TimeFormat(core.Now(), core.TimeRFC3339Nano)
	}
	ev.Meta = Sanitise(ev.Meta)

	line := core.JSONMarshalString(&ev) + "\n"

	s.mu.Lock()
	defer s.mu.Unlock()

	// Fs.Append creates the parent directory and the file when absent.
	r := s.fs.Append(s.path)
	if !r.OK {
		core.Error("audit: open append failed", "path", s.path, "err", r.Value)
		return
	}
	if w := core.WriteAll(r.Value, line); !w.OK {
		core.Error("audit: write failed", "path", s.path, "err", w.Value)
	}
}
