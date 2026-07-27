// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for the pure value-coercion + classification helpers
// scattered across the agentic command surface. These are deterministic,
// no-I/O functions whose switch arms went largely unexercised; each test
// drives every branch with a representative input.

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// --- stateValueString (commands_state.go) ----------------------------

func TestHelpers_stateValueString_AllBranches(t *testing.T) {
	// string passes through verbatim.
	core.AssertEqual(t, "hello", stateValueString("hello"))
	// a marshalable value renders as JSON.
	core.AssertEqual(t, `{"a":1}`, stateValueString(map[string]any{"a": 1}))
	// a slice renders as a JSON array.
	core.AssertEqual(t, `[1,2]`, stateValueString([]int{1, 2}))
}

// --- brainListStringValue (commands.go) ------------------------------

func TestHelpers_brainListStringValue_AllBranches(t *testing.T) {
	core.AssertEqual(t, "hi", brainListStringValue("hi"))
	core.AssertEqual(t, "7", brainListStringValue(7))
	core.AssertEqual(t, "8", brainListStringValue(int64(8)))
	core.AssertEqual(t, "9", brainListStringValue(float64(9)))
	// unhandled type yields empty.
	core.AssertEqual(t, "", brainListStringValue([]string{"x"}))
}

// --- flowStepDisplayName (flow.go) -----------------------------------

func TestHelpers_flowStepDisplayName_Precedence(t *testing.T) {
	// Name wins.
	core.AssertEqual(t, "build", flowStepDisplayName(0, flowDefinitionStep{Name: "build", Cmd: "go"}))
	// Cmd is next.
	core.AssertEqual(t, "go test", flowStepDisplayName(1, flowDefinitionStep{Cmd: "go test"}))
	// Flow is next.
	core.AssertEqual(t, "deploy", flowStepDisplayName(2, flowDefinitionStep{Flow: "deploy"}))
	// Run is next.
	core.AssertEqual(t, "echo hi", flowStepDisplayName(3, flowDefinitionStep{Run: "echo hi"}))
	// All empty falls back to a positional name.
	core.AssertEqual(t, "step-4", flowStepDisplayName(4, flowDefinitionStep{}))
}

// --- planRetentionDays (plan_retention.go) ---------------------------

func TestHelpers_planRetentionDays_OptionTypes(t *testing.T) {
	core.AssertEqual(t, 5, planRetentionDays(core.NewOptions(core.Option{Key: "days", Value: 5})))
	core.AssertEqual(t, 6, planRetentionDays(core.NewOptions(core.Option{Key: "days", Value: int64(6)})))
	core.AssertEqual(t, 7, planRetentionDays(core.NewOptions(core.Option{Key: "days", Value: float64(7)})))
	core.AssertEqual(t, 8, planRetentionDays(core.NewOptions(core.Option{Key: "days", Value: "8"})))
}

func TestHelpers_planRetentionDays_EnvFallback(t *testing.T) {
	// No --days option, but the env var is set.
	t.Setenv("AGENTIC_PLAN_RETENTION_DAYS", "30")
	core.AssertEqual(t, 30, planRetentionDays(core.NewOptions()))
}

func TestHelpers_planRetentionDays_Default(t *testing.T) {
	t.Setenv("AGENTIC_PLAN_RETENTION_DAYS", "")
	// Empty string option + no env → the package default.
	core.AssertEqual(t, planRetentionDefaultDays,
		planRetentionDays(core.NewOptions(core.Option{Key: "days", Value: "   "})))
}

// --- optionStrings (process_register.go) -----------------------------

func TestHelpers_optionStrings_AllBranches(t *testing.T) {
	// Missing key → nil.
	core.AssertNil(t, optionStrings(core.NewOptions(), "tags"))
	// []string passes through.
	core.AssertEqual(t, []string{"a", "b"},
		optionStrings(core.NewOptions(core.Option{Key: "tags", Value: []string{"a", "b"}}), "tags"))
	// []any is coerced element-wise.
	core.AssertEqual(t, []string{"1", "x"},
		optionStrings(core.NewOptions(core.Option{Key: "tags", Value: []any{1, "x"}}), "tags"))
	// scalar is wrapped in a single-element slice.
	core.AssertEqual(t, []string{"solo"},
		optionStrings(core.NewOptions(core.Option{Key: "tags", Value: "solo"}), "tags"))
}

// --- pipelineAuditIssueType / pipelineAuditSeverity (pipeline_audit.go)

func TestHelpers_pipelineAuditIssueType_Classification(t *testing.T) {
	cases := []struct {
		title, body, want string
	}{
		{"Fix auth bypass", "owasp", "security"},
		{"Add missing test coverage", "", "testing"},
		{"Improve performance", "perf hot path", "performance"},
		{"Update docs", "documentation", "docs"},
		{"Refactor module", "tidy up", "quality"},
	}
	for _, tc := range cases {
		got := pipelineAuditIssueType(pipelineIssueRecord{Title: tc.title, Body: tc.body})
		core.AssertEqual(t, tc.want, got)
	}
}

func TestHelpers_pipelineAuditIssueType_LabelSignal(t *testing.T) {
	// The classifier also folds label names into the haystack.
	issue := pipelineIssueRecord{
		Title:  "generic title",
		Labels: []pipelineLabelRecord{{Name: "security"}},
	}
	core.AssertEqual(t, "security", pipelineAuditIssueType(issue))
}

func TestHelpers_pipelineAuditSeverity_Classification(t *testing.T) {
	cases := []struct{ title, want string }{
		{"critical RCE", "critical"},
		{"high severity leak", "high"},
		{"medium issue", "medium"},
		{"low priority nit", "low"},
		{"unlabelled", ""},
	}
	for _, tc := range cases {
		core.AssertEqual(t, tc.want, pipelineAuditSeverity(pipelineIssueRecord{Title: tc.title}))
	}
}

// --- pipelineDisplayTheme (pipeline_commands.go) ---------------------

func TestHelpers_pipelineDisplayTheme_AllThemes(t *testing.T) {
	cases := map[string]string{
		"security":    "Security",
		"testing":     "Testing",
		"docs":        "Docs",
		"performance": "Performance",
		"features":    "Features",
		"anything":    "Quality",
	}
	for in, want := range cases {
		core.AssertEqual(t, want, pipelineDisplayTheme(in))
	}
}

// --- pipelineEpicTheme (pipeline_epic.go) ----------------------------

func TestHelpers_pipelineEpicTheme_Classification(t *testing.T) {
	core.AssertEqual(t, "security", pipelineEpicTheme(PipelineIssueRef{Title: "security hardening"}))
	core.AssertEqual(t, "testing", pipelineEpicTheme(PipelineIssueRef{Title: "add tests"}))
	core.AssertEqual(t, "docs", pipelineEpicTheme(PipelineIssueRef{Title: "doc sweep"}))
	core.AssertEqual(t, "performance", pipelineEpicTheme(PipelineIssueRef{Title: "perf pass"}))
	core.AssertEqual(t, "features", pipelineEpicTheme(PipelineIssueRef{Title: "feat(api): new endpoint"}))
	core.AssertEqual(t, "quality", pipelineEpicTheme(PipelineIssueRef{Title: "tidy"}))
	// Labels also feed the haystack.
	core.AssertEqual(t, "security",
		pipelineEpicTheme(PipelineIssueRef{Title: "x", Labels: []string{"security"}}))
}

// --- contentSchemaItemMap (content.go) -------------------------------

func TestHelpers_contentSchemaItemMap_AllBranches(t *testing.T) {
	// map passes through.
	m := map[string]any{"k": "v"}
	core.AssertEqual(t, m, contentSchemaItemMap(m))
	// typed question.
	q := contentSchemaItemMap(ContentSchemaQuestion{Question: "Q?", Answer: "A"})
	core.AssertEqual(t, "Q?", q["question"])
	core.AssertEqual(t, "A", q["answer"])
	// typed step.
	s := contentSchemaItemMap(ContentSchemaStep{Name: "n", Text: "t", URL: "u"})
	core.AssertEqual(t, "n", s["name"])
	core.AssertEqual(t, "u", s["url"])
	// JSON string is parsed.
	j := contentSchemaItemMap(`{"question":"hi"}`)
	core.AssertEqual(t, "hi", j["question"])
	// empty string → nil.
	core.AssertNil(t, contentSchemaItemMap("   "))
	// unhandled type → nil.
	core.AssertNil(t, contentSchemaItemMap(42))
}

// --- contentSchemaItemsValue (content.go) ----------------------------

func TestHelpers_contentSchemaItemsValue_AllBranches(t *testing.T) {
	// typed question slice.
	qs := contentSchemaItemsValue([]ContentSchemaQuestion{{Question: "Q", Answer: "A"}})
	core.AssertLen(t, qs, 1)
	core.AssertEqual(t, "Q", qs[0]["question"])
	// typed step slice.
	ss := contentSchemaItemsValue([]ContentSchemaStep{{Name: "n", Text: "t"}})
	core.AssertLen(t, ss, 1)
	core.AssertEqual(t, "n", ss[0]["name"])
	// []map passes through.
	ms := contentSchemaItemsValue([]map[string]any{{"a": 1}})
	core.AssertLen(t, ms, 1)
	// []any of mixed maps.
	as := contentSchemaItemsValue([]any{map[string]any{"x": 1}})
	core.AssertLen(t, as, 1)
	// single map → one-element slice.
	one := contentSchemaItemsValue(map[string]any{"only": true})
	core.AssertLen(t, one, 1)
	// JSON array string.
	arr := contentSchemaItemsValue(`[{"a":1},{"b":2}]`)
	core.AssertLen(t, arr, 2)
	// JSON object string.
	obj := contentSchemaItemsValue(`{"a":1}`)
	core.AssertLen(t, obj, 1)
	// empty string → nil.
	core.AssertNil(t, contentSchemaItemsValue(""))
	// unhandled type → nil.
	core.AssertNil(t, contentSchemaItemsValue(3.14))
}

// --- fetchLoopDuration (fetch_loop.go) -------------------------------

func TestHelpers_fetchLoopDuration_AllBranches(t *testing.T) {
	core.AssertEqual(t, 5*time.Second, fetchLoopDuration(5*time.Second))
	core.AssertEqual(t, 90*time.Second, fetchLoopDuration("90s"))
	core.AssertEqual(t, 3*time.Second, fetchLoopDuration(3))
	core.AssertEqual(t, 4*time.Second, fetchLoopDuration(int64(4)))
	core.AssertEqual(t, 2*time.Second, fetchLoopDuration(float64(2)))
	// non-positive / unparsable / unhandled → 0.
	core.AssertEqual(t, time.Duration(0), fetchLoopDuration(0))
	core.AssertEqual(t, time.Duration(0), fetchLoopDuration("not-a-duration"))
	core.AssertEqual(t, time.Duration(0), fetchLoopDuration(struct{}{}))
}

// --- phaseValue / phaseDependenciesValue (plan.go) -------------------

func TestHelpers_phaseValue_AllBranches(t *testing.T) {
	// already a Phase.
	p, ok := phaseValue(Phase{Number: 1, Name: "design"})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "design", p.Name)

	// map form.
	p, ok = phaseValue(map[string]any{
		"number":      2,
		"name":        "build",
		"description": "do the thing",
		"status":      "active",
		"criteria":    []any{"a", "b"},
	})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 2, p.Number)
	core.AssertEqual(t, "build", p.Name)
	core.AssertLen(t, p.Criteria, 2)

	// JSON string form.
	p, ok = phaseValue(`{"number":3,"name":"ship"}`)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ship", p.Name)

	// non-JSON string → not a phase.
	_, ok = phaseValue("not-json")
	core.AssertFalse(t, ok)

	// unhandled type → not a phase.
	_, ok = phaseValue(123)
	core.AssertFalse(t, ok)
}

func TestHelpers_phaseDependenciesValue_AllBranches(t *testing.T) {
	core.AssertEqual(t, []string{"x", "y"}, phaseDependenciesValue([]string{"x", "y"}))
	core.AssertEqual(t, []string{"a", "b"}, phaseDependenciesValue([]any{"a", " b "}))
	// []any with a non-string element bails to nil.
	core.AssertNil(t, phaseDependenciesValue([]any{"a", 1}))
	// JSON-array string is parsed.
	core.AssertEqual(t, []string{"p", "q"}, phaseDependenciesValue(`["p","q"]`))
	// comma-separated string is split + cleaned.
	core.AssertEqual(t, []string{"m", "n"}, phaseDependenciesValue("m, n"))
	// empty string → nil.
	core.AssertNil(t, phaseDependenciesValue("   "))
	// a non-collection scalar is coerced to a single-element slice.
	core.AssertEqual(t, []string{"42"}, phaseDependenciesValue(42))
}
