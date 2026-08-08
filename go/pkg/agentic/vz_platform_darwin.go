// SPDX-License-Identifier: EUPL-1.2

//go:build darwin

package agentic

import (
	core "dappco.re/go"
	"dappco.re/go/container"
)

// Everything go-container gates behind //go:build darwin is reached through
// this file, so the rest of pkg/agentic compiles on every platform. vz.go in
// dappco.re/go/container carries that constraint, which puts IsVZAvailable,
// NewVZProvider and ExecResult out of reach of a Linux build — the package
// referenced all three unguarded and so failed to build in CI while building
// fine on a developer's Mac.

// vzHostAvailable reports whether Virtualization.framework is usable here.
//
//	vzHostAvailable() // true on an Apple-silicon host with the classes resolved
func vzHostAvailable() bool { return container.IsVZAvailable() }

// newVZProviderImpl returns the concrete in-process VZ provider.
//
//	provider := newVZProviderImpl()
func newVZProviderImpl() vzDispatcher { return container.NewVZProvider() }

// vzDecodeExec normalises an ExecResult verb payload into the platform-neutral
// vzExec the fork works with.
//
// container.ExecResult is checked second so an injected vzExec still decodes:
// the unit tests script the fake dispatcher with vzExec directly rather than
// with a type they cannot name on Linux.
//
//	exec, ok := vzDecodeExec(provider.ExecResult(id, "sh", "-c", "true"))
func vzDecodeExec(result core.Result) (vzExec, bool) {
	if exec, ok := result.Value.(vzExec); ok {
		return exec, true
	}

	native, ok := result.Value.(container.ExecResult)
	if !ok {
		return vzExec{}, false
	}

	return vzExec{Stdout: native.Stdout, Stderr: native.Stderr, Exit: native.Exit}, true
}
