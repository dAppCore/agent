// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	agentpkg "dappco.re/go/agent"
	"github.com/stretchr/testify/assert"
)

func TestUpdate_UpdateChannel_Good(t *testing.T) {
	agentpkg.Version = "1.0.0"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	assert.Equal(t, "stable", updateChannel())
}

func TestUpdate_UpdateChannelDev_Good(t *testing.T) {
	agentpkg.Version = "dev"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	assert.Equal(t, "dev", updateChannel())
}

func TestUpdate_UpdateChannelEmpty_Bad(t *testing.T) {
	agentpkg.Version = ""
	assert.Equal(t, "dev", updateChannel())
}

func TestUpdate_UpdateChannelPrerelease_Ugly(t *testing.T) {
	agentpkg.Version = "0.8.0-alpha"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	assert.Equal(t, "prerelease", updateChannel())
}

func TestUpdate_UpdateChannelNumericSuffix_Ugly(t *testing.T) {
	agentpkg.Version = "0.8.0-beta.1"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	// Ends in '1' which is < 'a', so stable
	assert.Equal(t, "stable", updateChannel())
}

func TestUpdate_AppVersion_Good(t *testing.T) {
	agentpkg.Version = "1.2.3"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})
	assert.Equal(t, "1.2.3", appVersion())
}

func TestUpdate_AppVersion_Bad(t *testing.T) {
	agentpkg.Version = ""
	assert.Equal(t, "dev", appVersion())
}
