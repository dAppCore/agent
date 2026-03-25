// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// --- parseForgeArgs ---

func TestParseForgeArgs_Good_AllFields(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "myorg"},
		core.Option{Key: "_arg", Value: "myrepo"},
		core.Option{Key: "number", Value: "42"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "myorg", org)
	assert.Equal(t, "myrepo", repo)
	assert.Equal(t, int64(42), num)
}

func TestParseForgeArgs_Good_DefaultOrg(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "core", org, "should default to 'core'")
	assert.Equal(t, "go-io", repo)
	assert.Equal(t, int64(0), num, "no number provided")
}

func TestParseForgeArgs_Bad_EmptyOpts(t *testing.T) {
	opts := core.NewOptions()
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "core", org, "should default to 'core'")
	assert.Empty(t, repo)
	assert.Equal(t, int64(0), num)
}

func TestParseForgeArgs_Bad_InvalidNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "repo"},
		core.Option{Key: "number", Value: "not-a-number"},
	)
	_, _, num := parseForgeArgs(opts)
	assert.Equal(t, int64(0), num, "invalid number should parse as 0")
}

// --- fmtIndex ---

func TestFmtIndex_Good(t *testing.T) {
	assert.Equal(t, "1", fmtIndex(1))
	assert.Equal(t, "42", fmtIndex(42))
	assert.Equal(t, "0", fmtIndex(0))
	assert.Equal(t, "999999", fmtIndex(999999))
}
