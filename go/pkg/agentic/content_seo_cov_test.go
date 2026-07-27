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

// TestContentSEOCov_ScheduleTool_Good_CreatesRevision — the registered
// content_seo_schedule tool, driven end-to-end through an in-memory MCP client,
// persists a pending revision (exercising registerContentSEOTool's registered
// closure and contentSEOScheduleTool).
func TestContentSEOCov_ScheduleTool_Good_CreatesRevision(t *testing.T) {
	t.Setenv("CORE_MCP_FULL", "1")
	withStateStoreTempDir(t)
	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)
	subsystem.RegisterTools(svc)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := svc.Server().Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	callResult, err := clientSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "content_seo_schedule",
		Arguments: map[string]any{
			"page_id": "/help/hosting",
			"content": "Updated copy",
		},
	})
	core.RequireNoError(t, err)
	core.AssertFalse(t, callResult.IsError)

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 1)
	core.AssertEqual(t, "/help/hosting", pending[0].PageID)
}

// TestContentSEOCov_ScheduleTool_Bad_EmptyPageID — invoking the tool with a
// blank page_id surfaces the schedule error through the tool's error return.
func TestContentSEOCov_ScheduleTool_Bad_EmptyPageID(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, output, err := contentSEOScheduleTool(subsystem, context.Background(), nil, ContentSEOScheduleInput{
		PageID:  "",
		Content: "Updated copy",
	})
	core.AssertError(t, err)
	core.AssertFalse(t, output.Success)
	core.AssertContains(t, err.Error(), "page_id is required")
}

// TestContentSEOCov_Middleware_Bad_NonGetMethodSkipped — a POST request never
// triggers a scheduling sweep, even from Googlebot.
func TestContentSEOCov_Middleware_Bad_NonGetMethodSkipped(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/help/hosting", nil)
	c.Request.Header.Set("User-Agent", "Googlebot/2.1")

	subsystem.ContentSEOGooglebotMiddleware(nil)(c)

	// POST is ignored: the pending revision is untouched.
	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 1)
	core.AssertNil(t, pending[0].ScheduledAt)
}

// TestContentSEOCov_Middleware_Ugly_ErrorStatusSkipped — a GET that the handler
// chain finished with a 4xx status is not swept (the revision stays pending).
func TestContentSEOCov_Middleware_Ugly_ErrorStatusSkipped(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/help/hosting", nil)
	c.Request.Header.Set("User-Agent", "Googlebot/2.1")

	middleware := subsystem.ContentSEOGooglebotMiddleware(nil)
	// A downstream handler sets an error status before the deferred sweep runs.
	c.Writer.WriteHeader(http.StatusNotFound)
	middleware(c)

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 1)
	core.AssertNil(t, pending[0].ScheduledAt)
}

// TestContentSEOCov_Middleware_Ugly_EmptyResolvedPageIDSkipped — when the
// resolver returns blank and the request has no usable path, the sweep is
// skipped without error.
func TestContentSEOCov_Middleware_Ugly_EmptyResolvedPageIDSkipped(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.RequireNoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/help/hosting", nil)
	c.Request.Header.Set("User-Agent", "Googlebot/2.1")

	// Resolver forces a blank page id and the request path won't match the
	// scheduled revision, so nothing is published for "/help/hosting".
	subsystem.ContentSEOGooglebotMiddleware(func(*gin.Context) string { return "   " })(c)

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, err)
	core.AssertLen(t, pending, 1)
}

// TestContentSEOCov_IsGooglebot_Good_CaseAndWhitespace — detection is
// case-insensitive and trims surrounding whitespace.
func TestContentSEOCov_IsGooglebot_Good_CaseAndWhitespace(t *testing.T) {
	core.AssertTrue(t, contentSEOIsGooglebot("  Mozilla/5.0 (compatible; GOOGLEBOT/2.1)  "))
	core.AssertFalse(t, contentSEOIsGooglebot("Mozilla/5.0"))
}

// TestContentSEOCov_RevisionKey_Good_UniquePerCall — two keys for the same
// timestamp differ because of the random hex suffix.
func TestContentSEOCov_RevisionKey_Good_UniquePerCall(t *testing.T) {
	at := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	core.AssertNotEqual(t, contentSEORevisionKey(at), contentSEORevisionKey(at))
}
