// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- normaliseSessionAgentType (colon forms + non-claude reject) ---

func TestSession_NormaliseSessionAgentType_Good_ColonForms(t *testing.T) {
	for input, want := range map[string]string{
		"claude:opus":   "opus",
		"claude:sonnet": "sonnet",
		"claude:haiku":  "haiku",
	} {
		got, ok := normaliseSessionAgentType(input)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, want, got)
	}
}

func TestSession_NormaliseSessionAgentType_Good_BareAliases(t *testing.T) {
	got, ok := normaliseSessionAgentType("claude")
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "opus", got)

	got, ok = normaliseSessionAgentType("haiku")
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "haiku", got)
}

func TestSession_NormaliseSessionAgentType_Bad_NonClaudeColonRejected(t *testing.T) {
	// A colon form whose prefix is not claude is rejected.
	got, ok := normaliseSessionAgentType("gpt:4")
	core.AssertFalse(t, ok)
	core.AssertEmpty(t, got)

	// A claude colon form with an unknown model is rejected.
	got, ok = normaliseSessionAgentType("claude:ultra")
	core.AssertFalse(t, ok)
	core.AssertEmpty(t, got)
}

// --- storeSession (writes cache; round-trips via readSessionCache) ---

func TestSession_StoreSession_Good_PersistsAndMerges(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())

	s := newPrepWithProcess()
	stored := s.storeSession(Session{
		SessionID: "sess-store-1",
		PlanSlug:  "core/go-io",
		AgentType: "opus",
		Status:    "active",
	})

	core.AssertEqual(t, "sess-store-1", stored.SessionID)
	core.AssertNotEmpty(t, stored.CreatedAt)
	core.AssertNotEmpty(t, stored.UpdatedAt)

	// The cache file is on disk and a second store merges existing fields.
	core.AssertTrue(t, fs.IsFile(sessionCachePath("sess-store-1")).OK)

	merged := s.storeSession(Session{SessionID: "sess-store-1", Summary: "all done"})
	core.AssertEqual(t, "opus", merged.AgentType) // inherited from the first store
	core.AssertEqual(t, "all done", merged.Summary)
}

func TestSession_StoreSession_Bad_MissingSessionIDReturnsInput(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())

	s := newPrepWithProcess()
	// mergeSessionCache errors on an empty SessionID; storeSession returns the
	// (unmerged) input rather than panicking.
	in := Session{AgentType: "opus"}
	out := s.storeSession(in)
	core.AssertEqual(t, "opus", out.AgentType)
	core.AssertEmpty(t, out.SessionID)
}

func TestSession_StoreSession_Ugly_WriteFailureReturnsMerged(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())

	// Force the cache write to fail; storeSession returns the merged session
	// (not the raw input) so the caller still sees the resolved fields.
	orig := writeSessionCache
	t.Cleanup(func() { writeSessionCache = orig })
	writeSessionCache = func(_ *Session) error {
		return core.E("writeSessionCache", "disk full", nil)
	}

	s := newPrepWithProcess()
	out := s.storeSession(Session{SessionID: "sess-write-fail", AgentType: "opus"})
	core.AssertEqual(t, "sess-write-fail", out.SessionID)
	// merge stamped CreatedAt/UpdatedAt even though the write failed.
	core.AssertNotEmpty(t, out.UpdatedAt)
}

// --- sessionFromInput (empty-field fills) ---

func TestSession_SessionFromInput_Good_FillsEmptyFields(t *testing.T) {
	got := sessionFromInput(Session{}, SessionStartInput{
		PlanSlug:  "core/go-io",
		AgentType: "opus",
		Context:   map[string]any{"repo": "go-io"},
	})
	core.AssertEqual(t, "core/go-io", got.PlanSlug)
	core.AssertEqual(t, "core/go-io", got.Plan)
	core.AssertEqual(t, "opus", got.AgentType)
	core.AssertEqual(t, "go-io", stringValue(got.ContextSummary["repo"]))
}

func TestSession_SessionFromInput_Ugly_PreservesExisting(t *testing.T) {
	// Pre-set fields are not overwritten by the input.
	got := sessionFromInput(Session{
		PlanSlug:       "kept/plan",
		Plan:           "kept/plan",
		AgentType:      "sonnet",
		ContextSummary: map[string]any{"k": "v"},
	}, SessionStartInput{PlanSlug: "new/plan", AgentType: "opus", Context: map[string]any{"x": "y"}})
	core.AssertEqual(t, "kept/plan", got.PlanSlug)
	core.AssertEqual(t, "sonnet", got.AgentType)
	core.AssertEqual(t, "v", stringValue(got.ContextSummary["k"]))
}

// --- sessionEndFromInput (terminal status sets EndedAt; handoff merge) ---

func TestSession_SessionEndFromInput_Good_TerminalSetsEndedAt(t *testing.T) {
	got := sessionEndFromInput(Session{SessionID: "s1"}, SessionEndInput{
		Status:  "completed",
		Summary: "wrapped up",
		Handoff: map[string]any{"summary": "carry on"},
	})
	core.AssertEqual(t, "completed", got.Status)
	core.AssertEqual(t, "wrapped up", got.Summary)
	core.AssertNotEmpty(t, got.EndedAt)
	core.AssertEqual(t, "carry on", stringValue(got.Handoff["summary"]))
}

func TestSession_SessionEndFromInput_Ugly_HandoffNotesFallback(t *testing.T) {
	// When Handoff is empty but HandoffNotes is set, notes become the handoff.
	got := sessionEndFromInput(Session{SessionID: "s2"}, SessionEndInput{
		Status:       "handed_off",
		HandoffNotes: map[string]any{"next_steps": []any{"do x"}},
	})
	core.AssertNotEmpty(t, got.Handoff)
	core.AssertNotEmpty(t, got.EndedAt)
}

func TestSession_SessionEndFromInput_Bad_NonTerminalNoEndedAt(t *testing.T) {
	// A non-terminal status leaves EndedAt empty.
	got := sessionEndFromInput(Session{SessionID: "s3"}, SessionEndInput{Status: "active"})
	core.AssertEmpty(t, got.EndedAt)
}

// --- sessionBrainProject (3 return paths) ---

func TestSession_SessionBrainProject_Good_FromContextSummary(t *testing.T) {
	project := sessionBrainProject(
		Session{ContextSummary: map[string]any{"repo": "go-io"}},
		map[string]any{"repo": "ignored"},
	)
	core.AssertEqual(t, "go-io", project)
}

func TestSession_SessionBrainProject_Ugly_FromContextForNext(t *testing.T) {
	// Falls back to context_for_next when the session summary has no repo.
	project := sessionBrainProject(Session{}, map[string]any{"repo": "go-scm"})
	core.AssertEqual(t, "go-scm", project)
}

func TestSession_SessionBrainProject_Bad_Empty(t *testing.T) {
	core.AssertEmpty(t, sessionBrainProject(Session{}, nil))
}

// --- sessionProgressSummary (message fallback + Unknown) ---

func TestSession_SessionProgressSummary_Good_FromAction(t *testing.T) {
	summary := sessionProgressSummary([]map[string]any{
		{"type": "checkpoint", "action": "ran tests", "timestamp": "t1"},
		{"type": "error", "action": "build failed", "timestamp": "t2"},
	})
	core.AssertEqual(t, 2, summary["completed_steps"])
	core.AssertEqual(t, 1, summary["checkpoint_count"])
	core.AssertEqual(t, 1, summary["error_count"])
	core.AssertEqual(t, "build failed", summary["last_action"])
}

func TestSession_SessionProgressSummary_Ugly_MessageFallbackAndUnknown(t *testing.T) {
	// No action key → falls back to message.
	withMessage := sessionProgressSummary([]map[string]any{{"message": "did a thing"}})
	core.AssertEqual(t, "did a thing", withMessage["last_action"])

	// Neither action nor message → "Unknown".
	unknown := sessionProgressSummary([]map[string]any{{"type": "note"}})
	core.AssertEqual(t, "Unknown", unknown["last_action"])
}

func TestSession_SessionProgressSummary_Bad_Empty(t *testing.T) {
	summary := sessionProgressSummary(nil)
	core.AssertEqual(t, 0, summary["completed_steps"])
	core.AssertEqual(t, "No work recorded", summary["summary"])
	core.AssertNil(t, summary["last_action"])
}

// --- sessionDataMap (nested envelope + flat fallback) ---

func TestSession_SessionDataMap_Good_NestedEnvelope(t *testing.T) {
	data := sessionDataMap(map[string]any{
		"session": map[string]any{"id": 7, "status": "active"},
	})
	core.AssertEqual(t, 7, intValue(data["id"]))
	core.AssertEqual(t, "active", stringValue(data["status"]))
}

func TestSession_SessionDataMap_Bad_FlatFallback(t *testing.T) {
	// No nested "session" key → the payload itself is returned.
	payload := map[string]any{"id": 9, "status": "done"}
	data := sessionDataMap(payload)
	core.AssertEqual(t, 9, intValue(data["id"]))
}

func TestSession_SessionDataMap_Ugly_ResourceEmptyReturnsPayload(t *testing.T) {
	// An error-only payload yields no resource map, so sessionDataMap falls
	// through to returning the original payload unchanged.
	payload := map[string]any{"error": "boom"}
	data := sessionDataMap(payload)
	core.AssertEqual(t, "boom", stringValue(data["error"]))
}

// --- sessionHandoffMemoryContent (full content with all sections) ---

func TestSession_SessionHandoffMemoryContent_Good_FullContent(t *testing.T) {
	content := sessionHandoffMemoryContent(
		Session{SessionID: "s9", PlanSlug: "core/go-io", AgentType: "opus", Status: "handed_off"},
		"summary text",
		[]string{"step one", "step two"},
		[]string{"blocker one"},
		map[string]any{"repo": "go-io"},
	)
	core.AssertContains(t, content, "Session handoff: s9")
	core.AssertContains(t, content, "Plan: core/go-io")
	core.AssertContains(t, content, "Agent: opus")
	core.AssertContains(t, content, "Status: handed_off")
	core.AssertContains(t, content, "summary text")
	core.AssertContains(t, content, "- step one")
	core.AssertContains(t, content, "- blocker one")
	core.AssertContains(t, content, "Context for next:")
}

func TestSession_SessionHandoffMemoryContent_Bad_Minimal(t *testing.T) {
	// Only the session id — optional sections are omitted.
	content := sessionHandoffMemoryContent(Session{SessionID: "s10"}, "", nil, nil, nil)
	core.AssertContains(t, content, "Session handoff: s10")
	core.AssertNotContains(t, content, "Next steps:")
	core.AssertNotContains(t, content, "Blockers:")
}

// --- sessionHandoffMemoryTags (clean + plan slug) ---

func TestSession_SessionHandoffMemoryTags_Good_IncludesAgentAndPlan(t *testing.T) {
	tags := sessionHandoffMemoryTags(Session{AgentType: "opus", PlanSlug: "core/go-io"})
	core.AssertContains(t, tags, "session")
	core.AssertContains(t, tags, "handoff")
	core.AssertContains(t, tags, "opus")
	core.AssertContains(t, tags, "core/go-io")
}
