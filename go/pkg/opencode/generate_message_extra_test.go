// SPDX-Licence-Identifier: EUPL-1.2

// sendMessage arm coverage. generate_test.go covers the session-create
// failure + the happy path + the no-text degenerate reply; this file fills
// the message-step arms the happy path skips: the agent payload branch, a
// transport error on the message POST, and an upstream 4xx/5xx on the
// message POST. Reuses the fakeCall + withFakeSandbox seam from
// generate_test.go (no live container).

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestGenerate_sendMessage_AgentBranch_Good — a non-empty Agent populates
// the "agent" key in the message payload (the input.Agent != "" arm). The
// recorded request body carries both the agent and the prompt.
func TestGenerate_sendMessage_AgentBranch_Good(t *testing.T) {
	fc := &fakeCall{
		sessionBody: `{"id":"ses-agent"}`,
		sessionCode: 200,
		messageBody: `{"parts":[{"type":"text","text":"done"}]}`,
		messageCode: 200,
	}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{
			Prompt: "review this diff",
			Agent:  "code-review",
		})
	})

	core.AssertTrue(t, got.OK)
	core.AssertEqual(t, "done", got.Value)
	// The message body (second call) carries the agent + the prompt.
	core.AssertEqual(t, 2, len(fc.bodies))
	core.AssertContains(t, fc.bodies[1], "code-review")
	core.AssertContains(t, fc.bodies[1], "review this diff")
}

// TestGenerate_sendMessage_Bad_TransportError — the message POST returns a
// transport error (messageErr): Generate fails with the send-message
// failure, distinct from the session-create error path.
func TestGenerate_sendMessage_Bad_TransportError(t *testing.T) {
	fc := &fakeCall{
		sessionBody: `{"id":"ses-x"}`,
		sessionCode: 200,
		messageErr:  core.E("opencode.test", "dial tcp refused", nil),
	}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{Prompt: "hi"})
	})

	core.AssertFalse(t, got.OK)
	core.AssertContains(t, got.Error(), "send message failed")
	// The session was created (1) then the message POST attempted (2).
	core.AssertEqual(t, 2, len(fc.urls))
}

// TestGenerate_sendMessage_Bad_UpstreamStatus — the message POST returns
// 502: Generate fails and the error carries the status + body (the message
// code >= 400 arm).
func TestGenerate_sendMessage_Bad_UpstreamStatus(t *testing.T) {
	fc := &fakeCall{
		sessionBody: `{"id":"ses-y"}`,
		sessionCode: 200,
		messageBody: "upstream exploded",
		messageCode: 502,
	}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{Prompt: "hi"})
	})

	core.AssertFalse(t, got.OK)
	core.AssertContains(t, got.Error(), "502")
	core.AssertContains(t, got.Error(), "upstream exploded")
}
