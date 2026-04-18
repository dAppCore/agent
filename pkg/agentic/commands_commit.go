// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func (s *PrepSubsystem) registerCommitCommands() {
	c := s.Core()
	c.Command("commit", core.Command{Description: "Write the final dispatch record to the workspace journal", Action: s.cmdCommit})
	c.Command("agentic:commit", core.Command{Description: "Write the final dispatch record to the workspace journal", Action: s.cmdCommit})
}

// core-agent commit core/go-io/task-42
func (s *PrepSubsystem) cmdCommit(options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	if workspace == "" {
		core.Print(nil, "usage: core-agent commit <workspace>")
		return core.Result{Value: core.E("agentic.cmdCommit", "workspace is required", nil), OK: false}
	}

	result := s.handleCommit(s.commandContext(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspace},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdCommit", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(CommitOutput)
	if !ok {
		err := core.E("agentic.cmdCommit", "invalid commit output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "workspace: %s", output.Workspace)
	core.Print(nil, "journal:   %s", output.JournalPath)
	if output.MarkerPath != "" {
		core.Print(nil, "marker:    %s", output.MarkerPath)
	}
	if output.Skipped {
		core.Print(nil, "skipped:   true")
	} else {
		core.Print(nil, "committed: %s", output.CommittedAt)
	}
	return core.Result{Value: output, OK: true}
}
