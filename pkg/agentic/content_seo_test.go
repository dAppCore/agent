// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/gin-gonic/gin"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPendingRevision_PrepSubsystem_ScheduleRevision_Good(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	revision, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "/help/hosting", revision.PageID)
	core.AssertEqual(t, "Updated copy", revision.Content)
	core.AssertNil(t, revision.ScheduledAt)
	core.AssertTrue(t, revision.CreatedAt.Equal(now))

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 1)
	core.AssertNil(t, pending[0].ScheduledAt)

	var rawEntries []string
	subsystem.stateStoreRestore(contentSEORevisionGroup, func(_ string, value string) bool {
		rawEntries = append(rawEntries, value)
		return true
	})
	core.AssertLen(t, rawEntries, 1)
	core.AssertContains(t, rawEntries[0], `"scheduled_at":null`)
}

func TestEmptyPage_PrepSubsystem_ScheduleRevision_Bad(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "", "Updated copy")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "page_id is required")
}

func TestPublishWindow_PrepSubsystem_OnGooglebotVisit_Good(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)
	restoreContentSEORandomDelay(t, 37*time.Minute)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)
	core.RequireNoError(t, subsystem.OnGooglebotVisit(context.Background(), "/help/hosting"))

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 0)

	records, err := subsystem.contentSEORevisionRecords(subsystem.stateStoreInstance(), "/help/hosting", false)
	core.RequireNoError(t, err)
	core.AssertLen(t, records, 1)
	core.AssertNotNil(t, records[0].Revision.ScheduledAt)

	delta := records[0].Revision.ScheduledAt.Sub(now)
	core.AssertGreaterOrEqual(t, delta, 8*time.Minute)
	core.AssertLessOrEqual(t, delta, 62*time.Minute)
	core.AssertEqual(t, 37*time.Minute, delta)
}

func TestNoPendingRevision_PrepSubsystem_OnGooglebotVisit_Bad(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	core.RequireNoError(t, subsystem.OnGooglebotVisit(context.Background(), "/help/hosting"))
	core.AssertEqual(t, 0, subsystem.stateStoreCount(contentSEORevisionGroup))
}

func TestStoreError_PrepSubsystem_OnGooglebotVisit_Ugly(t *testing.T) {
	root := t.TempDir()
	blocked := root + "/blocked"
	writeResult := fs.Write(blocked, "blocked")
	core.RequireTrue(t, writeResult.OK)
	t.Setenv("CORE_WORKSPACE", blocked)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	err := subsystem.OnGooglebotVisit(context.Background(), "/help/hosting")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "state store unavailable")
}

func TestPendingList_PrepSubsystem_GetPendingRevisions_Good(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 1)
	core.AssertEqual(t, "/help/hosting", pending[0].PageID)
}

func TestEmptyPage_PrepSubsystem_GetPendingRevisions_Bad(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	pending, err := subsystem.GetPendingRevisions("")
	core.AssertError(t, err)
	core.AssertNil(t, pending)
}

func TestScheduledFiltered_PrepSubsystem_GetPendingRevisions_Ugly(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)
	restoreContentSEORandomDelay(t, 37*time.Minute)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)
	core.RequireNoError(t, subsystem.OnGooglebotVisit(context.Background(), "/help/hosting"))

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 0)
}

func TestGooglebot_PrepSubsystem_HandleGooglebotVisit_Good(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)
	restoreContentSEORandomDelay(t, 37*time.Minute)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)
	core.RequireNoError(t, subsystem.HandleGooglebotVisit(context.Background(), "/help/hosting", "Googlebot/2.1"))

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 0)
}

func TestNonGooglebot_PrepSubsystem_HandleGooglebotVisit_Bad(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	err := subsystem.HandleGooglebotVisit(context.Background(), "/help/hosting", "Mozilla/5.0")
	core.RequireNoError(t, err)
	core.AssertEqual(t, 0, subsystem.stateStoreCount(contentSEORevisionGroup))
}

func TestEmptyPage_PrepSubsystem_HandleGooglebotVisit_Ugly(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	err := subsystem.HandleGooglebotVisit(context.Background(), "", "Googlebot/2.1")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "page_id is required")
}

func TestContentSEO_RegisterTools_Good_RegistersScheduleTool(t *testing.T) {
	t.Setenv("CORE_MCP_FULL", "1")

	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	core.AssertContains(t, toolNames, "content_seo_schedule")
}

func TestGooglebotMiddleware_PrepSubsystem_ContentSEOGooglebotMiddleware_Good(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)
	restoreContentSEORandomDelay(t, 37*time.Minute)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/help/hosting", nil)
	c.Request.Header.Set("User-Agent", "Googlebot/2.1")

	subsystem.ContentSEOGooglebotMiddleware(nil)(c)

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 0)
}

func TestNonGooglebotMiddleware_PrepSubsystem_ContentSEOGooglebotMiddleware_Bad(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/help/hosting", nil)
	c.Request.Header.Set("User-Agent", "Mozilla/5.0")

	core.AssertNotPanics(t, func() {
		subsystem.ContentSEOGooglebotMiddleware(nil)(c)
	})
	core.AssertEqual(t, 0, subsystem.stateStoreCount(contentSEORevisionGroup))
}

func TestResolvedPage_PrepSubsystem_ContentSEOGooglebotMiddleware_Ugly(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)
	restoreContentSEORandomDelay(t, 37*time.Minute)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/resolved/page", "Updated copy")
	core.RequireNoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ignored", nil)
	c.Request.Header.Set("User-Agent", "Googlebot/2.1")

	subsystem.ContentSEOGooglebotMiddleware(func(*gin.Context) string { return "/resolved/page" })(c)

	pending, err := subsystem.GetPendingRevisions("/resolved/page")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 0)
}

func restoreContentSEONow(t *testing.T, now time.Time) {
	t.Helper()

	previous := contentSEONow
	contentSEONow = func() time.Time { return now }
	t.Cleanup(func() {
		contentSEONow = previous
	})
}

func restoreContentSEORandomDelay(t *testing.T, delay time.Duration) {
	t.Helper()

	previous := contentSEORandomDelay
	contentSEORandomDelay = func() (time.Duration, error) { return delay, nil }
	t.Cleanup(func() {
		contentSEORandomDelay = previous
	})
}
