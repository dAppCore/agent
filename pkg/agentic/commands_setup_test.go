// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/setup"
)

func TestCommandsSetup_CmdSetup_Good_WritesCoreConfigs(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	c := core.New(core.WithService(setup.Register))
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.cmdSetup(core.NewOptions(core.Option{Key: "path", Value: dir}))
	core.RequireTrue(t, result.OK)

	build := fs.Read(core.JoinPath(dir, ".core", "build.yaml"))
	core.RequireTrue(t, build.OK)
	core.AssertContains(t, build.Value.(string), "type: go")
}

func TestCommandsSetup_CmdSetup_Bad_MissingService(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.cmdSetup(core.NewOptions())
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "setup service is required")
}

func TestCommandsSetup_CmdSetup_Ugly_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

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
	core.RequireTrue(t, result.OK)
	core.AssertFalse(t, fs.Exists(core.JoinPath(dir, ".core")))
	core.AssertFalse(t, fs.Exists(core.JoinPath(dir, "PROMPT.md")))
}

func TestCommandsSetup_HandleSetup_Good_ActionAlias(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	c := core.New(core.WithService(setup.Register))
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.handleSetup(context.Background(), core.NewOptions(core.Option{Key: "path", Value: dir}))
	core.RequireTrue(t, result.OK)

	createdPath, ok := result.Value.(string)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, dir, createdPath)
	core.AssertTrue(t, fs.Exists(core.JoinPath(dir, ".core", "build.yaml")))
}
