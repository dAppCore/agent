// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestPlanCov_PhaseCriteriaList_Good_MergesAndDeduplicates — when both Criteria
// and CompletionCriteria are populated they merge with duplicates removed (the
// dedup-merge branch that neither early-return reaches).
func TestPlanCov_PhaseCriteriaList_Good_MergesAndDeduplicates(t *testing.T) {
	phase := Phase{
		Criteria:           []string{"tests pass", "lint clean"},
		CompletionCriteria: []string{"tests pass", "docs updated"},
	}

	merged := phaseCriteriaList(phase)

	core.AssertEqual(t, []string{"tests pass", "lint clean", "docs updated"}, merged)
}

// TestPlanCov_PhaseCriteriaList_Good_EachEmptySideFallsBack — an empty criteria
// side returns the other side unchanged (both early-return arms).
func TestPlanCov_PhaseCriteriaList_Good_EachEmptySideFallsBack(t *testing.T) {
	core.AssertEqual(t, []string{"only completion"},
		phaseCriteriaList(Phase{CompletionCriteria: []string{"only completion"}}))
	core.AssertEqual(t, []string{"only criteria"},
		phaseCriteriaList(Phase{Criteria: []string{"only criteria"}}))
}

// TestPlanCov_PhaseSliceValue_Ugly_UnknownTypeIsNil — an unrecognised value
// type that is not a single phase yields nil (the terminal fall-through).
func TestPlanCov_PhaseSliceValue_Ugly_UnknownTypeIsNil(t *testing.T) {
	core.AssertNil(t, phaseSliceValue(42))
	// A non-bracket string is also not a phase slice.
	core.AssertNil(t, phaseSliceValue("not a json array"))
}

// TestPlanCov_PhaseSliceValue_Good_JSONStringOfObjects — a JSON-array string of
// phase objects decodes through the string branch.
func TestPlanCov_PhaseSliceValue_Good_JSONStringOfObjects(t *testing.T) {
	phases := phaseSliceValue(`[{"name":"Setup","status":"pending"},{"name":"Build"}]`)

	core.AssertLen(t, phases, 2)
	core.AssertEqual(t, "Setup", phases[0].Name)
	core.AssertEqual(t, "Build", phases[1].Name)
}

// TestPlanCov_HandlePlanCreate_Bad_MissingTitle — the plan.create action
// wrapper surfaces the validation error when no title is supplied (the error
// arm of the wrapper).
func TestPlanCov_HandlePlanCreate_Bad_MissingTitle(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	result := s.handlePlanCreate(context.Background(), core.NewOptions(
		core.Option{Key: "objective", Value: "no title supplied"},
	))

	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.RequireTrue(t, ok)
}

// TestPlanCov_HandlePlanRead_Good_ReturnsPlan — the plan.read action wrapper
// returns the typed read output for an existing plan id (the success arm).
func TestPlanCov_HandlePlanRead_Good_ReturnsPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Readable Plan",
		Objective: "Read me back via the action wrapper",
	})
	core.RequireNoError(t, err)

	result := s.handlePlanRead(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: created.ID},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(PlanReadOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, created.ID, output.Plan.ID)
	core.AssertEqual(t, "Readable Plan", output.Plan.Title)
}

// TestPlanCov_WritePlanResult_Bad_NilPlan — a nil plan is rejected with the
// "plan is required" envelope before any filesystem work.
func TestPlanCov_WritePlanResult_Bad_NilPlan(t *testing.T) {
	result := writePlanResult(t.TempDir(), nil)

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "plan is required")
}

// TestPlanCov_WritePlanResult_Ugly_EnsureDirFailsUnderFile — pointing the plans
// directory at a path whose parent is a regular file makes EnsureDir fail, so
// the "failed to create plans directory" arm is taken.
func TestPlanCov_WritePlanResult_Ugly_EnsureDirFailsUnderFile(t *testing.T) {
	base := t.TempDir()
	filePath := core.JoinPath(base, "blocker")
	core.RequireTrue(t, fs.Write(filePath, "not a directory").OK)

	// blocker is a file; treating it as a parent directory must fail.
	result := writePlanResult(core.JoinPath(filePath, "plans"), &Plan{ID: "id-1-aaaaaa", Title: "Blocked"})

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "failed to create plans directory")
}

// TestPlanCov_WritePlanResult_Good_ReturnsPath — a valid plan writes to disk and
// returns the JSON path.
func TestPlanCov_WritePlanResult_Good_ReturnsPath(t *testing.T) {
	dir := t.TempDir()

	result := writePlanResult(dir, &Plan{ID: "id-7-abcdef", Title: "Write Me", Status: "draft"})

	core.RequireTrue(t, result.OK)
	path, ok := result.Value.(string)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, core.JoinPath(dir, "id-7-abcdef.json"), path)
	core.AssertTrue(t, fs.IsFile(path))
}

// TestPlanCov_CleanPlanSlug_Good_NormalisesSeparators — assorted separators
// collapse to single dashes and leading/trailing dashes are trimmed.
func TestPlanCov_CleanPlanSlug_Good_NormalisesSeparators(t *testing.T) {
	core.AssertEqual(t, "ax-rfc-follow-up", cleanPlanSlug("  AX/RFC__follow .up  "))
	core.AssertEqual(t, "a-b", cleanPlanSlug("a---b"))
	// Trailing separator survives normalisation then the trailing-dash trim removes it.
	core.AssertEqual(t, "a-b", cleanPlanSlug("a.b."))
}

// TestPlanCov_CleanPlanSlug_Bad_EmptyAndInvalid — an empty value and the literal
// "invalid" both clean to the empty string (the reserved-word and empty arms).
func TestPlanCov_CleanPlanSlug_Bad_EmptyAndInvalid(t *testing.T) {
	core.AssertEqual(t, "", cleanPlanSlug(""))
	core.AssertEqual(t, "", cleanPlanSlug("   "))
	core.AssertEqual(t, "", cleanPlanSlug("invalid"))
	// A string of only separators collapses to empty after trimming dashes.
	core.AssertEqual(t, "", cleanPlanSlug("///"))
}

// TestPlanCov_PlanSlugValue_Good_FallsBackToTitleAndSuffix — with no explicit
// slug, the title is cleaned and the id's last segment is appended as a suffix.
func TestPlanCov_PlanSlugValue_Good_FallsBackToTitleAndSuffix(t *testing.T) {
	core.AssertEqual(t, "my-plan-abc123", planSlugValue("", "My Plan", "id-42-abc123"))
}

// TestPlanCov_PlanSlugValue_Ugly_BlankTitleUsesPlanBase — a blank title falls
// back to the "plan" base before the suffix.
func TestPlanCov_PlanSlugValue_Ugly_BlankTitleUsesPlanBase(t *testing.T) {
	core.AssertEqual(t, "plan-xyz", planSlugValue("", "   ", "id-1-xyz"))
}

// TestPlanCov_PlanSlugSuffix_Good_LastSegment — the suffix is the final
// dash-delimited segment of the id.
func TestPlanCov_PlanSlugSuffix_Good_LastSegment(t *testing.T) {
	core.AssertEqual(t, "abc123", planSlugSuffix("id-42-abc123"))
}

// TestPlanCov_PlanSlugSuffix_Ugly_EmptyId — an empty id yields an empty suffix
// (Split returns a single empty element, whose trim is "").
func TestPlanCov_PlanSlugSuffix_Ugly_EmptyId(t *testing.T) {
	core.AssertEqual(t, "", planSlugSuffix(""))
}

// TestPlanCov_FindPlanBySlugResult_Bad_BlankSlug — a blank slug short-circuits
// with the "plan not found: invalid" envelope before any glob.
func TestPlanCov_FindPlanBySlugResult_Bad_BlankSlug(t *testing.T) {
	result := findPlanBySlugResult(t.TempDir(), "   ")

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "plan not found: invalid")
}

// TestPlanCov_FindPlanBySlugResult_Ugly_NoMatchAfterScan — a non-empty plans
// directory with no matching slug walks every file then reports not-found.
func TestPlanCov_FindPlanBySlugResult_Ugly_NoMatchAfterScan(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, writePlanResult(dir, &Plan{ID: "id-1-aaaaaa", Slug: "alpha", Title: "Alpha"}).OK)
	core.RequireTrue(t, writePlanResult(dir, &Plan{ID: "id-2-bbbbbb", Slug: "beta", Title: "Beta"}).OK)
	// A stray non-JSON-decodable file is skipped, not fatal.
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "garbage.json"), "not json").OK)

	result := findPlanBySlugResult(dir, "gamma")

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "plan not found: gamma")
}

// TestPlanCov_FindPlanBySlugResult_Good_MatchBySlug — a matching slug returns
// the decoded plan pointer.
func TestPlanCov_FindPlanBySlugResult_Good_MatchBySlug(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, writePlanResult(dir, &Plan{ID: "id-3-cccccc", Slug: "delta", Title: "Delta"}).OK)

	result := findPlanBySlugResult(dir, "delta")

	core.RequireTrue(t, result.OK)
	plan, ok := result.Value.(*Plan)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "delta", plan.Slug)
	core.AssertEqual(t, "id-3-cccccc", plan.ID)
}
