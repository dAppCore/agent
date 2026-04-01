// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerSessionCommands() {
	c := s.Core()
	c.Command("session/resume", core.Command{Description: "Resume a paused or handed-off session from local cache", Action: s.cmdSessionResume})
	c.Command("session/replay", core.Command{Description: "Build replay context for a stored session", Action: s.cmdSessionReplay})
}

func (s *PrepSubsystem) cmdSessionResume(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session resume <session-id>")
		return core.Result{Value: core.E("agentic.cmdSessionResume", "session_id is required", nil), OK: false}
	}

	result := s.handleSessionResume(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionResume", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionResumeOutput)
	if !ok {
		err := core.E("agentic.cmdSessionResume", "invalid session resume output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session:    %s", output.Session.SessionID)
	core.Print(nil, "status:     %s", output.Session.Status)
	if len(output.HandoffContext) > 0 {
		core.Print(nil, "handoff:    %d item(s)", len(output.HandoffContext))
	}
	if len(output.RecentActions) > 0 {
		core.Print(nil, "recent:     %d action(s)", len(output.RecentActions))
	}
	if len(output.Artifacts) > 0 {
		core.Print(nil, "artifacts:  %d", len(output.Artifacts))
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSessionReplay(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session replay <session-id>")
		return core.Result{Value: core.E("agentic.cmdSessionReplay", "session_id is required", nil), OK: false}
	}

	result := s.handleSessionReplay(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionReplay", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionReplayOutput)
	if !ok {
		err := core.E("agentic.cmdSessionReplay", "invalid session replay output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", sessionID)
	core.Print(nil, "context items: %d", len(output.ReplayContext))
	return core.Result{Value: output, OK: true}
}
