// SPDX-License-Identifier: EUPL-1.2

package main

// version is set at build time via ldflags:
//
//	go build -ldflags "-X 'dappco.re/go/agent.version=0.15.0'" ./cmd/core-agent/
var version string

// updateChannel maps the build version to the release channel.
//
//	version = "0.15.0"
//	updateChannel() // "stable"
func updateChannel() string {
	switch {
	case version == "" || version == "dev":
		return "dev"
	case len(version) > 0 && (version[len(version)-1] >= 'a'):
		return "prerelease"
	default:
		return "stable"
	}
}

// TODO: wire go-update UpdateService for self-update command
// Channels: stable → GitHub releases, prerelease → GitHub dev, dev → Forge main
// Parked until version var moves to module root package (dappco.re/go/agent.Version)
