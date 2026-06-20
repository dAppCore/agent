// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for three otherwise-thin handlers:
//   - cmdPlanCleanup: the wrapper's disabled / no-match / dry-run / deleted
//     print branches (the underlying planCleanup is tested separately).
//   - contentBriefGet: guard + success-envelope + backend-error.
//   - findReviewCandidates: the local glob + non-repo-skip path (no git).

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- cmdPlanCleanup --------------------------------------------------

func TestMisc_CmdPlanCleanup_Disabled(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanCleanup(core.NewOptions(core.Option{Key: "days", Value: 0}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "disabled")
}

func TestMisc_CmdPlanCleanup_NoMatch(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanCleanup(core.NewOptions(core.Option{Key: "days", Value: 90}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "No plans found past the retention period.")
}

func TestMisc_CmdPlanCleanup_DryRun(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)
	_, err := writePlan(PlansRoot(), &Plan{
		ID: "stale-plan-abc123", Title: "Stale", Status: "archived",
		Objective: "old", ArchivedAt: time.Now().AddDate(0, 0, -120),
	})
	core.RequireNoError(t, err)

	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanCleanup(core.NewOptions(
			core.Option{Key: "days", Value: 90},
			core.Option{Key: "dry-run", Value: true},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "DRY RUN")
}

func TestMisc_CmdPlanCleanup_Deletes(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)
	_, err := writePlan(PlansRoot(), &Plan{
		ID: "gone-plan-abc123", Title: "Gone", Status: "archived",
		Objective: "old", ArchivedAt: time.Now().AddDate(0, 0, -120),
	})
	core.RequireNoError(t, err)

	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanCleanup(core.NewOptions(core.Option{Key: "days", Value: 90}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "deleted")
}

// --- contentBriefGet -------------------------------------------------

func TestMisc_ContentBriefGet_Bad_MissingID(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	r := s.contentBriefGet(context.Background(), ContentBriefGetInput{})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "brief_id is required")
}

func TestMisc_ContentBriefGet_Good_Envelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/briefs/brief-7", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"brief":{"id":7,"title":"Launch post","status":"ready"}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	r := s.contentBriefGet(context.Background(), ContentBriefGetInput{BriefID: "brief-7"})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(ContentBriefOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
}

func TestMisc_ContentBriefGet_Bad_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	r := s.contentBriefGet(context.Background(), ContentBriefGetInput{BriefID: "brief-7"})
	core.AssertFalse(t, r.OK)
}

// --- findReviewCandidates --------------------------------------------

// TestMisc_FindReviewCandidates_NoCandidates — a base path containing only
// non-git directories (and a file) yields no review candidates: the glob +
// IsDir filter + hasRemote-false skip path runs without any git remote.
func TestMisc_FindReviewCandidates_NoCandidates(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	base := t.TempDir()
	// A plain directory (no git remote) and a regular file.
	core.AssertTrue(t, fs.EnsureDir(core.JoinPath(base, "repo-a")).OK)
	core.AssertTrue(t, fs.Write(core.JoinPath(base, "not-a-dir.txt"), "x").OK)

	got := s.findReviewCandidates(base)
	core.AssertLen(t, got, 0)
}
