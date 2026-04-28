// SPDX-License-Identifier: EUPL-1.2

package main

import (
	core "dappco.re/go"
	agentpkg "dappco.re/go/agent"
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
