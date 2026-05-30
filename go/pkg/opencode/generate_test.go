// SPDX-License-Identifier: EUPL-1.2

package opencode

import (
	goio "io"
	"testing"

	core "dappco.re/go"
)

// fakeCall records the requests routed through callOpenCodeFn and
// replays scripted responses keyed by URL suffix, so the session/message
// flow runs without a live opencode-serve container.
type fakeCall struct {
	sessionBody string
	sessionCode int
	sessionErr  error

	messageBody string
	messageCode int
	messageErr  error

	urls    []string
	methods []string
	bodies  []string
}

func (f *fakeCall) handle(_ *Service, method, url string, body goio.Reader) (string, int, error) {
	raw, _ := goio.ReadAll(body)
	f.urls = append(f.urls, url)
	f.methods = append(f.methods, method)
	f.bodies = append(f.bodies, string(raw))

	if core.HasSuffix(url, "/session") {
		return f.sessionBody, f.sessionCode, f.sessionErr
	}
	return f.messageBody, f.messageCode, f.messageErr
}

// withFakeSandbox swaps the orm-backed sandbox-resolve + the HTTP client
// for the duration of fn, restoring the originals after.
func withFakeSandbox(fc *fakeCall, fn func()) {
	origEnsure := ensureSandboxFn
	origTarget := targetForFn
	origCall := callOpenCodeFn
	defer func() {
		ensureSandboxFn = origEnsure
		targetForFn = origTarget
		callOpenCodeFn = origCall
	}()
	ensureSandboxFn = func(_ *Service, _, _ string) core.Result { return core.Ok("oc-test") }
	targetForFn = func(_ *Service, _ string) (string, core.Result) {
		return "http://127.0.0.1:4096", core.Ok(nil)
	}
	callOpenCodeFn = fc.handle
	fn()
}

func TestGenerate_Generate_Good(t *testing.T) {
	fc := &fakeCall{
		sessionBody: `{"id":"ses-1"}`,
		sessionCode: 200,
		messageBody: `{"info":{"role":"assistant"},"parts":[{"type":"step-start"},{"type":"text","text":"Release "},{"type":"text","text":"note."}]}`,
		messageCode: 200,
	}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{
			Prompt:  "Draft a release note",
			Profile: "lemma",
			Model:   "core-local/lthn/lemma",
		})
	})

	core.AssertTrue(t, got.OK, "Generate should succeed")
	core.AssertEqual(t, "Release note.", got.Value)

	// Two calls: POST /session, then POST /session/ses-1/message.
	core.AssertEqual(t, 2, len(fc.urls))
	core.AssertContains(t, fc.urls[0], "/session")
	core.AssertContains(t, fc.urls[1], "/session/ses-1/message")
	core.AssertEqual(t, core.MethodPost, fc.methods[1])
	// The message body carries the prompt text + the pinned model.
	core.AssertContains(t, fc.bodies[1], "Draft a release note")
	core.AssertContains(t, fc.bodies[1], "core-local/lthn/lemma")
}

func TestGenerate_Generate_Bad_EmptyPrompt(t *testing.T) {
	got := (&Service{}).Generate(GenerateInput{Prompt: "   "})
	core.AssertTrue(t, !got.OK, "empty prompt should fail")
}

func TestGenerate_Generate_Bad_SessionUpstreamError(t *testing.T) {
	fc := &fakeCall{sessionBody: "boom", sessionCode: 500}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{Prompt: "hi"})
	})

	core.AssertTrue(t, !got.OK, "session 500 should fail")
	// No message call was attempted.
	core.AssertEqual(t, 1, len(fc.urls))
}

func TestGenerate_Generate_Ugly_NoTextPart(t *testing.T) {
	fc := &fakeCall{
		sessionBody: `{"id":"ses-2"}`,
		sessionCode: 200,
		// Only a tool part, no text — a degenerate-but-valid reply shape.
		messageBody: `{"parts":[{"type":"tool","text":""}]}`,
		messageCode: 200,
	}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{Prompt: "hi"})
	})

	core.AssertTrue(t, !got.OK, "reply with no text part should fail")
}

func TestGenerate_Generate_Ugly_SessionMissingID(t *testing.T) {
	fc := &fakeCall{sessionBody: `{"title":"untitled"}`, sessionCode: 200}

	var got core.Result
	withFakeSandbox(fc, func() {
		got = (&Service{}).Generate(GenerateInput{Prompt: "hi"})
	})

	core.AssertTrue(t, !got.OK, "session without id should fail")
}

func TestGenerate_extractMessageText_Good(t *testing.T) {
	body := `{"parts":[{"type":"text","text":"a"},{"type":"reasoning","text":"skip"},{"type":"text","text":"b"}]}`
	core.AssertEqual(t, "ab", extractMessageText(body))
}

func TestGenerate_extractMessageText_Bad_Malformed(t *testing.T) {
	core.AssertEqual(t, "", extractMessageText("not json"))
}
