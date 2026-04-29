// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	core "dappco.re/go"
	agentpkg "dappco.re/go/agent"
)

func TestUpdate_UpdateChannel_Good_Case(t *testing.T) {
	agentpkg.Version = "1.0.0"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	core.AssertEqual(t, "stable", updateChannel())
}

func TestUpdate_UpdateChannelDev_Good_Case(t *testing.T) {
	agentpkg.Version = "dev"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	core.AssertEqual(t, "dev", updateChannel())
}

func TestUpdate_UpdateChannelEmpty_Bad_Case(t *testing.T) {
	agentpkg.Version = ""
	got := updateChannel()
	core.AssertEqual(t, "dev", got)
	core.AssertNotEmpty(t, got)
}

func TestUpdate_UpdateChannelPrerelease_Ugly_Case(t *testing.T) {
	agentpkg.Version = "0.8.0-alpha"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	core.AssertEqual(t, "prerelease", updateChannel())
}

func TestUpdate_UpdateChannelNumericSuffix_Ugly_Case(t *testing.T) {
	agentpkg.Version = "0.8.0-beta.1"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	core.AssertEqual(t, "prerelease", updateChannel())
}

func TestUpdate_ApplicationVersion_Good_Case(t *testing.T) {
	agentpkg.Version = "1.2.3"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	core.AssertEqual(t, "1.2.3", applicationVersion())
}

func TestUpdate_ApplicationVersion_Bad_Case(t *testing.T) {
	agentpkg.Version = ""
	got := applicationVersion()
	core.AssertEqual(t, "dev", got)
	core.AssertNotEmpty(t, got)
}
