// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"os"
	"testing"

	"dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// newTestCore creates a minimal Core with app commands registered.
func newTestCore(t *testing.T) *core.Core {
	t.Helper()
	c := core.New(core.WithOption("name", "core-agent"))
	c.App().Version = "test"
	registerAppCommands(c)
	return c
}

// --- applyLogLevel ---

func TestCommands_ApplyLogLevel_Good(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()

	os.Args = []string{"core-agent", "--quiet", "version"}
	applyLogLevel()
	assert.Equal(t, []string{"core-agent", "version"}, os.Args)
}

func TestCommands_ApplyLogLevel_Good_Debug(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()

	os.Args = []string{"core-agent", "-d", "check"}
	applyLogLevel()
	assert.Equal(t, []string{"core-agent", "check"}, os.Args)
}

func TestCommands_ApplyLogLevel_Bad_NoFlag(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()

	os.Args = []string{"core-agent", "status"}
	applyLogLevel()
	assert.Equal(t, []string{"core-agent", "status"}, os.Args)
}

func TestCommands_ApplyLogLevel_Ugly_FlagAfterCommand(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()

	os.Args = []string{"core-agent", "version", "-q"}
	applyLogLevel()
	assert.Equal(t, []string{"core-agent", "version"}, os.Args)
}

// --- registerAppCommands ---

func TestCommands_RegisterAppCommands_Good(t *testing.T) {
	c := newTestCore(t)
	cmds := c.Commands()
	assert.Contains(t, cmds, "version")
	assert.Contains(t, cmds, "check")
	assert.Contains(t, cmds, "env")
}

// --- version command ---

func TestCommands_Version_Good(t *testing.T) {
	c := newTestCore(t)
	version = "0.8.0"

	r := c.Cli().Run("version")
	assert.True(t, r.OK)
}

func TestCommands_Version_Bad_DevVersion(t *testing.T) {
	c := newTestCore(t)
	version = ""
	c.App().Version = "dev"

	r := c.Cli().Run("version")
	assert.True(t, r.OK)
}

// --- check command ---

func TestCommands_Check_Good(t *testing.T) {
	c := newTestCore(t)

	r := c.Cli().Run("check")
	assert.True(t, r.OK)
}

// --- env command ---

func TestCommands_Env_Good(t *testing.T) {
	c := newTestCore(t)

	r := c.Cli().Run("env")
	assert.True(t, r.OK)
}

// --- CLI resolution ---

func TestCommands_Cli_Bad_UnknownCommand(t *testing.T) {
	c := newTestCore(t)
	r := c.Cli().Run("nonexistent")
	assert.False(t, r.OK)
}

func TestCommands_Cli_Good_Banner(t *testing.T) {
	c := newTestCore(t)
	c.Cli().SetBanner(func(_ *core.Cli) string {
		return "core-agent test"
	})
	// No args — shows banner, returns empty Result
	r := c.Cli().Run()
	_ = r
}

func TestCommands_Cli_Ugly_EmptyArgs(t *testing.T) {
	c := newTestCore(t)
	// Explicit empty slice
	r := c.Cli().Run()
	_ = r
}
