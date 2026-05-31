// SPDX-License-Identifier: EUPL-1.2

package opencode

import (
	core "dappco.re/go"
)

// AuditFunc is the hub-installable audit edge for this control surface.
// RFC.serve.md §7.3.1 — opencode runs inside a sandbox and does NOT
// audit itself; the no-op emit hooks in this package were only safe
// because the desktop (a SASE) audited at its access edge. The hub
// deletes that desktop edge and becomes the new edge, so it installs an
// AuditFunc via SetAuditSink and every privilege-bearing handler routes
// its already-redacted event through it.
//
// Implementations must be safe for concurrent calls and MUST NOT carry
// credential bytes (the emit-sites structurally cannot reach them; the
// hub's sink sanitises Meta defensively regardless).
//
// Usage example:
//
//	opencode.SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {
//	    sink.Emit(audit.Event{Event: event, Outcome: outcome, RequestID: requestID, Meta: meta})
//	})
type AuditFunc func(event, scope, outcome, requestID string, meta map[string]any)

// auditSink holds the installed edge. nil = no edge (CLI / stdio /
// serve modes where no hub is composing the route groups). Guarded by
// auditMu so SetAuditSink can be called after construction without
// racing the handlers.
var (
	auditMu   core.RWMutex
	auditSink AuditFunc
)

// SetAuditSink installs (or clears, with nil) the hub's audit edge. The
// hub calls this once at boot, after constructing its pkg/audit sink and
// before serving. Passing nil restores the no-op default.
//
// Usage example:
//
//	opencode.SetAuditSink(hubSink)
//	defer opencode.SetAuditSink(nil)
func SetAuditSink(fn AuditFunc) {
	auditMu.Lock()
	auditSink = fn
	auditMu.Unlock()
}

// dispatchAudit forwards a redacted event to the installed edge, if any.
// Called by emitControlAudit / emitPortAudit so the per-handler
// call-sites stay unchanged.
func dispatchAudit(event, scope, outcome, requestID string, meta map[string]any) {
	auditMu.RLock()
	fn := auditSink
	auditMu.RUnlock()
	if fn != nil {
		fn(event, scope, outcome, requestID, meta)
	}
}
