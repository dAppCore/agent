// SPDX-License-Identifier: EUPL-1.2

//go:build !darwin

package agentic

import (
	core "dappco.re/go"
	"dappco.re/go/container"
)

// The non-darwin half of the VZ seam. Virtualization.framework is Apple-only,
// so off darwin the host is simply never VZ-capable and the dispatcher exists
// only to satisfy the interface — vzDispatchEnabled() is false here, so nothing
// in the fork ever reaches it.

// vzHostAvailable reports whether Virtualization.framework is usable here.
// Never, off darwin.
//
//	vzHostAvailable() // false
func vzHostAvailable() bool { return false }

// newVZProviderImpl returns a dispatcher that reports itself unavailable.
//
//	provider := newVZProviderImpl()
func newVZProviderImpl() vzDispatcher { return vzUnsupportedProvider{} }

// vzDecodeExec normalises an ExecResult verb payload into vzExec.
//
// Only the injected form is decodable off darwin: container.ExecResult is
// declared inside go-container's darwin-tagged vz.go and cannot be named here.
// Production never reaches this — vzUnsupportedProvider fails before exec — but
// the unit tests script the fake dispatcher with vzExec and do run on Linux.
//
//	exec, ok := vzDecodeExec(core.Ok(vzExec{Stdout: "hi"}))
func vzDecodeExec(result core.Result) (vzExec, bool) {
	exec, ok := result.Value.(vzExec)

	return exec, ok
}

// vzUnsupportedProvider satisfies vzDispatcher on hosts with no
// Virtualization.framework, failing every verb rather than pretending.
type vzUnsupportedProvider struct{}

// Available always reports false off darwin.
func (vzUnsupportedProvider) Available() bool { return false }

// Run refuses: there is no framework to boot a guest with.
func (vzUnsupportedProvider) Run(_ *container.Image, _ ...container.RunOption) core.Result {
	return vzUnsupportedResult()
}

// ExecResult refuses: nothing can be running to exec into.
func (vzUnsupportedProvider) ExecResult(_, _ string, _ ...string) core.Result {
	return vzUnsupportedResult()
}

// Stop refuses: nothing can be running to stop.
func (vzUnsupportedProvider) Stop(_ string) core.Result { return vzUnsupportedResult() }

// vzUnsupportedResult is the single failure every verb returns here.
func vzUnsupportedResult() core.Result {
	return core.Fail(core.E("agentic.vz", "virtualization.framework is only available on darwin", nil))
}
