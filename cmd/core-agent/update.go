// SPDX-License-Identifier: EUPL-1.2

package main

import agentpkg "dappco.re/go/agent"

// updateChannel maps the build version to the release channel.
//
//	agentpkg.Version = "0.15.0"
//	updateChannel() // "stable"
func updateChannel() string {
	switch {
	case agentpkg.Version == "" || agentpkg.Version == "dev":
		return "dev"
	case agentpkg.Version != "" && (agentpkg.Version[len(agentpkg.Version)-1] >= 'a'):
		return "prerelease"
	default:
		return "stable"
	}
}
