// SPDX-License-Identifier: EUPL-1.2

package agent

// Version is injected at build time via ldflags.
//
//	go build -ldflags "-X 'dappco.re/go/agent.Version=0.15.0'" ./cmd/core-agent/
//	core.Println(Version) // "0.15.0"
var Version string
