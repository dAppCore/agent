// SPDX-License-Identifier: EUPL-1.2

package main

import (
	agentpkg "dappco.re/go/agent"
	core "dappco.re/go/core"
)

// agentpkg.Version = "0.15.0"
// updateChannel() // "stable"
func updateChannel() string {
	switch {
	case agentpkg.Version == "" || agentpkg.Version == "dev":
		return "dev"
	case core.Contains(agentpkg.Version, "-"):
		return "prerelease"
	default:
		return "stable"
	}
}
