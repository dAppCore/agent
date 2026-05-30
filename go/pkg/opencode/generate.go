// SPDX-Licence-Identifier: EUPL-1.2

// Generate — single-shot prompt → text completion against a sandboxed
// opencode-serve. core/agent OWNS opencode; the ProviderManager backend
// (pkg/agentic) drives generation through this method directly — no HTTP
// hop inside core/agent. The flow is opencode-serve's documented session
// API: POST /session creates a session, POST /session/:id/message sends
// the prompt and blocks for the assistant reply, and the assistant text
// is read out of the response parts.
//
// The sandbox boundary is indirected through callOpenCode (the same
// internal HTTP client ProviderList already uses), so unit tests fake
// opencode-serve without a live container by swapping callOpenCodeFn.

package opencode

import (
	goio "io"

	core "dappco.re/go"
)

const generateOp = "opencode.Generate"

// GenerateInput carries everything one Generate call needs. Profile
// selects the lthn-side opencode profile applied at spawn (and so the
// upstream provider + base model); Model optionally overrides the model
// id sent on the message (provider/model form, e.g. "core-local/lthn/
// lemma"); SandboxID optionally targets an already-running sandbox
// instead of ensuring one.
//
// Usage example:
//
//	r := svc.Generate(opencode.GenerateInput{Prompt: "Draft a release note", Profile: "gemma4-agentic"})
//	if r.OK { text := r.Value.(string); _ = text }
type GenerateInput struct {
	// Prompt is the user message text sent to the model. Required.
	Prompt string

	// Profile is the lthn-side opencode profile name applied when a new
	// sandbox is spawned. Empty falls back to DefaultProfile.
	Profile string

	// Model optionally overrides the message model id (provider/model
	// form). Empty lets opencode-serve use the profile's configured
	// default model.
	Model string

	// Agent optionally selects an opencode agent for the message.
	Agent string

	// SandboxID optionally targets a specific already-running sandbox.
	// Empty reuses the most-recent running sandbox or spawns one.
	SandboxID string
}

// Generate ensures a running opencode sandbox, creates a session, sends
// the prompt as a message, and returns the assistant's text reply.
//
// Synchronous — opencode-serve's /session/:id/message endpoint blocks
// until the model responds, so the returned text is complete on success.
//
// Usage example:
//
//	r := svc.Generate(opencode.GenerateInput{Prompt: "Summarise the diff", Profile: "lemma"})
//	if r.OK { reply := r.Value.(string); _ = reply }
func (s *Service) Generate(input GenerateInput) core.Result {
	if core.Trim(input.Prompt) == "" {
		return core.Fail(core.E(generateOp, "prompt is required", nil))
	}

	idR := ensureSandboxFn(s, input.SandboxID, input.Profile)
	if !idR.OK {
		return idR
	}
	id, _ := idR.Value.(string)

	target, r := targetForFn(s, id)
	if !r.OK {
		return r
	}

	sessionID, sr := s.createSession(target)
	if !sr.OK {
		return sr
	}

	return s.sendMessage(target, sessionID, input)
}

// ensureSandbox resolves a running sandbox to talk to. An explicit
// sandboxID must already be running. Otherwise the most-recent running
// sandbox is reused; if none is running, a new one is spawned with the
// requested profile.
func (s *Service) ensureSandbox(sandboxID, profile string) core.Result {
	sandboxID = core.Trim(sandboxID)
	if sandboxID != "" {
		if _, r := s.targetFor(sandboxID); !r.OK {
			return r
		}
		return core.Ok(sandboxID)
	}

	statusR := s.Status()
	if statusR.OK {
		if running, ok := statusR.Value.([]Sandbox); ok && len(running) > 0 {
			return core.Ok(running[0].ID)
		}
	}

	return s.Start(profile)
}

// createSession POSTs /session and returns the new session id.
func (s *Service) createSession(target string) (string, core.Result) {
	body, code, err := callOpenCodeFn(s, core.MethodPost, target+"/session", core.NewReader("{}"))
	if err != nil {
		return "", core.Fail(core.E(generateOp, "create session failed", err))
	}
	if code >= 400 {
		return "", core.Fail(core.E(generateOp,
			core.Sprintf("create session returned %d: %s", code, body), nil))
	}

	var session struct {
		ID string `json:"id"`
	}
	if ur := core.JSONUnmarshalString(body, &session); !ur.OK {
		return "", core.Fail(core.E(generateOp, core.Concat("decode session failed: ", ur.Error()), nil))
	}
	if core.Trim(session.ID) == "" {
		return "", core.Fail(core.E(generateOp, "session response carried no id", nil))
	}
	return session.ID, core.Ok(nil)
}

// sendMessage POSTs /session/:id/message and extracts the assistant
// text from the response parts.
func (s *Service) sendMessage(target, sessionID string, input GenerateInput) core.Result {
	payload := map[string]any{
		"parts": []map[string]any{
			{"type": "text", "text": input.Prompt},
		},
	}
	if core.Trim(input.Model) != "" {
		payload["model"] = input.Model
	}
	if core.Trim(input.Agent) != "" {
		payload["agent"] = input.Agent
	}

	body, code, err := callOpenCodeFn(s, core.MethodPost,
		target+"/session/"+sessionID+"/message", core.NewReader(core.JSONMarshalString(payload)))
	if err != nil {
		return core.Fail(core.E(generateOp, "send message failed", err))
	}
	if code >= 400 {
		return core.Fail(core.E(generateOp,
			core.Sprintf("send message returned %d: %s", code, body), nil))
	}

	text := extractMessageText(body)
	if core.Trim(text) == "" {
		return core.Fail(core.E(generateOp, "message response carried no text part", nil))
	}
	return core.Ok(text)
}

// extractMessageText pulls the concatenated text of every text part out
// of an opencode-serve /session/:id/message response ({info, parts}).
// Non-text parts (tool calls, step markers) are skipped.
func extractMessageText(body string) string {
	var resp struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if ur := core.JSONUnmarshalString(body, &resp); !ur.OK {
		return ""
	}

	builder := core.NewBuilder()
	for _, part := range resp.Parts {
		if part.Type == "text" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

// callOpenCodeFn indirects the internal HTTP client so unit tests fake
// opencode-serve without a live container. The default forwards to
// Service.callOpenCode (the same client ProviderList uses).
var callOpenCodeFn = func(s *Service, method, url string, body goio.Reader) (string, int, error) {
	return s.callOpenCode(method, url, body)
}

// ensureSandboxFn indirects sandbox resolution so unit tests exercise
// the session/message flow without an orm-backed Core or a live
// container. The default forwards to Service.ensureSandbox.
var ensureSandboxFn = func(s *Service, sandboxID, profile string) core.Result {
	return s.ensureSandbox(sandboxID, profile)
}

// targetForFn indirects sandbox-target resolution for the same reason.
// The default forwards to Service.targetFor.
var targetForFn = func(s *Service, id string) (string, core.Result) {
	return s.targetFor(id)
}
