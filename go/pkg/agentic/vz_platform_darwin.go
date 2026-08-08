// SPDX-License-Identifier: EUPL-1.2

//go:build darwin

package agentic

import (
	"dappco.re/go/container"
)

// The darwin half of the VZ seam. Virtualization.framework is Apple-only, so
// go-container declares IsVZAvailable and NewVZProvider behind //go:build
// darwin in vz.go; reaching them through this file is what keeps the rest of
// pkg/agentic compiling on every platform.

// vzHostAvailable reports whether Virtualization.framework is usable here.
//
//	vzHostAvailable() // true on an Apple-silicon host with the classes resolved
func vzHostAvailable() bool { return container.IsVZAvailable() }

// newVZProviderImpl returns the concrete in-process VZ provider.
//
//	provider := newVZProviderImpl()
func newVZProviderImpl() vzDispatcher { return container.NewVZProvider() }
