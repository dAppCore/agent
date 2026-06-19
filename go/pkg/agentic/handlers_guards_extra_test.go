// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestAgenticHandlers_IdentifierGuards — sprint + content handlers that require
// an identifier reject empty input before any platform call (the mock platform
// guarantees no real network is touched).
func TestAgenticHandlers_IdentifierGuards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.sprintGet(ctx, SprintGetInput{}).OK)
		core.AssertFalse(t, s.sprintStart(ctx, SprintTransitionInput{}).OK)
		core.AssertFalse(t, s.sprintComplete(ctx, SprintTransitionInput{}).OK)
		core.AssertFalse(t, s.sprintArchive(ctx, SprintArchiveInput{}).OK)
		core.AssertFalse(t, s.contentBriefGet(ctx, ContentBriefGetInput{}).OK)
	})
}

// TestAgenticHandlers_SessionIssueGuards — session + issue handlers that require
// an identifier reject empty input before any platform call.
func TestAgenticHandlers_SessionIssueGuards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.sessionGet(ctx, SessionGetInput{}).OK)
		core.AssertFalse(t, s.sessionEnd(ctx, SessionEndInput{}).OK)
		core.AssertFalse(t, s.sessionContinue(ctx, SessionContinueInput{}).OK)
		core.AssertFalse(t, s.sessionResume(ctx, SessionResumeInput{}).OK)
		core.AssertFalse(t, s.sessionLog(ctx, SessionLogInput{}).OK)
		core.AssertFalse(t, s.sessionArtifact(ctx, SessionArtifactInput{}).OK)
		core.AssertFalse(t, s.sessionHandoff(ctx, SessionHandoffInput{}).OK)
		core.AssertFalse(t, s.sessionReplay(ctx, SessionReplayInput{}).OK)
		core.AssertFalse(t, s.issueUpdate(ctx, IssueUpdateInput{}).OK)
		core.AssertFalse(t, s.issueComment(ctx, IssueCommentInput{}).OK)
		core.AssertFalse(t, s.issueArchive(ctx, IssueArchiveInput{}).OK)
	})
}

// TestAgenticHandlers_ContentGuards — content generate/create handlers reject
// empty input.
func TestAgenticHandlers_ContentGuards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.contentGenerate(ctx, ContentGenerateInput{}).OK)
		core.AssertFalse(t, s.contentBatchGenerate(ctx, ContentBatchGenerateInput{}).OK)
		core.AssertFalse(t, s.contentBriefCreate(ctx, ContentBriefCreateInput{}).OK)
		core.AssertFalse(t, s.contentFromPlan(ctx, ContentFromPlanInput{}).OK)
	})
}

// TestAgenticHandlers_ListCreate_Exercised — the remaining list/create platform
// handlers run their request path; an unparseable platform response makes each
// fail rather than succeed (mock → no real network).
func TestAgenticHandlers_ListCreate_Exercised(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.sprintCreate(ctx, SprintCreateInput{}).OK)
		core.AssertFalse(t, s.sprintList(ctx, SprintListInput{}).OK)
		core.AssertFalse(t, s.sessionStart(ctx, SessionStartInput{}).OK)
		core.AssertFalse(t, s.sessionList(ctx, SessionListInput{}).OK)
		core.AssertFalse(t, s.contentBriefList(ctx, ContentBriefListInput{}).OK)
		core.AssertFalse(t, s.contentStatus(ctx, ContentStatusInput{}).OK)
		core.AssertFalse(t, s.contentUsageStats(ctx, ContentUsageStatsInput{}).OK)
	})
}
