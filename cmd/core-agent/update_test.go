// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdate_UpdateChannel_Good(t *testing.T) {
	version = "1.0.0"
	assert.Equal(t, "stable", updateChannel())
}

func TestUpdate_UpdateChannel_Good_Dev(t *testing.T) {
	version = "dev"
	assert.Equal(t, "dev", updateChannel())
}

func TestUpdate_UpdateChannel_Good_Empty(t *testing.T) {
	version = ""
	assert.Equal(t, "dev", updateChannel())
}

func TestUpdate_UpdateChannel_Good_Prerelease(t *testing.T) {
	version = "0.8.0-alpha"
	assert.Equal(t, "prerelease", updateChannel())
}

func TestUpdate_UpdateChannel_Ugly(t *testing.T) {
	version = "0.8.0-beta.1"
	// Ends in '1' which is < 'a', so stable
	assert.Equal(t, "stable", updateChannel())
}

func TestUpdate_AppVersion_Good(t *testing.T) {
	version = "1.2.3"
	assert.Equal(t, "1.2.3", appVersion())
}

func TestUpdate_AppVersion_Good_Empty(t *testing.T) {
	version = ""
	assert.Equal(t, "dev", appVersion())
}
