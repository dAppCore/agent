// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	"dappco.re/go/agent/pkg/setup"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsSetup_CmdSetup_Good_WritesCoreConfigs(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	c := core.New(core.WithService(setup.Register))
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.cmdSetup(core.NewOptions(core.Option{Key: "path", Value: dir}))
	require.True(t, result.OK)

	build := fs.Read(core.JoinPath(dir, ".core", "build.yaml"))
	require.True(t, build.OK)
	assert.Contains(t, build.Value.(string), "type: go")
}

func TestCommandsSetup_CmdSetup_Bad_MissingService(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.cmdSetup(core.NewOptions())
	require.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "setup service is required")
}

func TestCommandsSetup_CmdSetup_Ugly_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	c := core.New(core.WithService(setup.Register))
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.cmdSetup(core.NewOptions(
		core.Option{Key: "path", Value: dir},
		core.Option{Key: "dry-run", Value: true},
		core.Option{Key: "template", Value: "agent"},
	))
	require.True(t, result.OK)
	assert.False(t, fs.Exists(core.JoinPath(dir, ".core")))
	assert.False(t, fs.Exists(core.JoinPath(dir, "PROMPT.md")))
}
