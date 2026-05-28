// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"net/http"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	store "dappco.re/go/store"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Usage example: `revision := agentic.SEORevision{PageID: "/help/hosting", Content: "Updated copy", ScheduledAt: nil, CreatedAt: time.Now()}`
type SEORevision struct {
	PageID      string     `json:"page_id"`
	Content     string     `json:"content"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// input := agentic.ContentSEOScheduleInput{PageID: "/help/hosting", Content: "Updated copy"}
type ContentSEOScheduleInput struct {
	PageID  string `json:"page_id"`
	Content string `json:"content"`
}

// out := agentic.ContentSEOScheduleOutput{Success: true, Revision: agentic.SEORevision{PageID: "/help/hosting"}}
type ContentSEOScheduleOutput struct {
	Success  bool        `json:"success"`
	Revision SEORevision `json:"revision"`
}

type seoRevisionRecord struct {
	Key      string
	Revision SEORevision
}

const contentSEORevisionGroup = "seo_revisions"
const contentSEOPageIDRequired = "page_id is required"

var (
	contentSEONow         = time.Now
	contentSEORandomDelay = func() (time.Duration, error) {
		window := big.NewInt(55)
		value, err := rand.Int(rand.Reader, window)
		if err != nil {
			return 0, core.E("contentSEORandomDelay", "read random delay", err)
		}
		return time.Duration(value.Int64()+8) * time.Minute, nil
	}
)

func (s *PrepSubsystem) registerContentSEOTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "content_seo_schedule",
		Description: "Create a pending Natural Progression SEO revision that stays unpublished until a Googlebot visit schedules it.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input ContentSEOScheduleInput) (*mcp.CallToolResult, ContentSEOScheduleOutput, error) {
		return contentSEOScheduleTool(s, ctx, request, input)
	})
}

var contentSEOScheduleTool = func(s *PrepSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input ContentSEOScheduleInput) (*mcp.CallToolResult, ContentSEOScheduleOutput, error) {
	revision, err := ScheduleRevision(s, ctx, input.PageID, input.Content)
	if err != nil {
		return nil, ContentSEOScheduleOutput{}, err
	}

	return nil, ContentSEOScheduleOutput{
		Success:  true,
		Revision: revision,
	}, nil
}

// revision, err := subsystem.ScheduleRevision(ctx, "/help/hosting", "Updated copy")
var ScheduleRevision = func(s *PrepSubsystem, ctx context.Context, pageID, content string) (SEORevision, error) {
	if err := contentSEOContextErr("scheduleRevision", ctx); err != nil {
		return SEORevision{}, err
	}

	pageID = core.Trim(pageID)
	if pageID == "" {
		return SEORevision{}, core.E("scheduleRevision", contentSEOPageIDRequired, nil)
	}

	storeInstance, err := contentSEOStore(s)
	if err != nil {
		return SEORevision{}, err
	}

	revision := SEORevision{
		PageID:      pageID,
		Content:     content,
		ScheduledAt: nil,
		CreatedAt:   contentSEONow(),
	}
	if result := storeInstance.Set(contentSEORevisionGroup, contentSEORevisionKey(revision.CreatedAt), core.JSONMarshalString(revision)); !result.OK {
		return SEORevision{}, core.E("scheduleRevision", "persist revision", resultErrorValue("scheduleRevision", result))
	}

	return revision, nil
}

// revisions, err := subsystem.GetPendingRevisions("/help/hosting")
var GetPendingRevisions = func(s *PrepSubsystem, pageID string) ([]SEORevision, error) {
	pageID = core.Trim(pageID)
	if pageID == "" {
		return nil, core.E("getPendingRevisions", contentSEOPageIDRequired, nil)
	}

	storeInstance, err := contentSEOStore(s)
	if err != nil {
		return nil, err
	}

	records, err := contentSEORevisionRecords(s, storeInstance, pageID, true)
	if err != nil {
		return nil, err
	}

	revisions := make([]SEORevision, 0, len(records))
	for _, record := range records {
		revisions = append(revisions, record.Revision)
	}
	return revisions, nil
}

// err := subsystem.OnGooglebotVisit(ctx, "/help/hosting")
var OnGooglebotVisit = func(s *PrepSubsystem, ctx context.Context, pageID string) error {
	if err := contentSEOContextErr("onGooglebotVisit", ctx); err != nil {
		return err
	}

	pageID = core.Trim(pageID)
	if pageID == "" {
		return core.E("onGooglebotVisit", contentSEOPageIDRequired, nil)
	}

	storeInstance, err := contentSEOStore(s)
	if err != nil {
		return err
	}

	records, err := contentSEORevisionRecords(s, storeInstance, pageID, true)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	baseTime := contentSEONow()
	if result := storeInstance.Transaction(func(transaction *store.StoreTransaction) core.Result {
		for _, record := range records {
			if err := contentSEOContextErr("onGooglebotVisit", ctx); err != nil {
				return core.Fail(err)
			}

			delay, err := contentSEORandomDelay()
			if err != nil {
				return core.Fail(core.E("onGooglebotVisit", "compute publish delay", err))
			}

			scheduledAt := baseTime.Add(delay)
			record.Revision.ScheduledAt = &scheduledAt
			if result := transaction.Set(contentSEORevisionGroup, record.Key, core.JSONMarshalString(record.Revision)); !result.OK {
				return core.Fail(core.E("onGooglebotVisit", "persist scheduled revision", resultErrorValue("onGooglebotVisit", result)))
			}
		}
		return core.Ok(nil)
	}); !result.OK {
		return core.E("onGooglebotVisit", "transaction", resultErrorValue("onGooglebotVisit", result))
	}

	return nil
}

// err := subsystem.HandleGooglebotVisit(ctx, "/help/hosting", request.UserAgent())
var HandleGooglebotVisit = func(s *PrepSubsystem, ctx context.Context, pageID, userAgent string) error {
	if !contentSEOIsGooglebot(userAgent) {
		return nil
	}
	return OnGooglebotVisit(s, ctx, pageID)
}

// middleware := subsystem.ContentSEOGooglebotMiddleware(nil)
func (s *PrepSubsystem) ContentSEOGooglebotMiddleware(resolvePageID func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c == nil || c.Request == nil || c.Request.Method != http.MethodGet {
			return
		}
		if !contentSEOIsGooglebot(c.Request.UserAgent()) {
			return
		}
		if c.Writer != nil && c.Writer.Status() >= http.StatusBadRequest {
			return
		}

		pageID := ""
		if resolvePageID != nil {
			pageID = resolvePageID(c)
		}
		if pageID == "" && c.Request.URL != nil {
			pageID = c.Request.URL.Path
		}
		pageID = core.Trim(pageID)
		if pageID == "" {
			return
		}

		if err := OnGooglebotVisit(s, c.Request.Context(), pageID); err != nil {
			core.Warn("content seo googlebot trigger failed", "page_id", pageID, "error", err)
		}
	}
}

var contentSEOStore = func(s *PrepSubsystem) (*store.Store, error) {
	if s == nil {
		return nil, core.E("contentSEOStore", "subsystem is nil", nil)
	}

	storeInstance := s.stateStoreInstance()
	if storeInstance != nil {
		return storeInstance, nil
	}
	if err := stateStoreErr(s); err != nil {
		return nil, core.E("contentSEOStore", "state store unavailable", err)
	}
	return nil, core.E("contentSEOStore", "state store unavailable", nil)
}

var contentSEORevisionRecords = func(_ *PrepSubsystem, storeInstance *store.Store, pageID string, pendingOnly bool) ([]seoRevisionRecord, error) {
	pageID = core.Trim(pageID)
	records := make([]seoRevisionRecord, 0)

	for entry, err := range storeInstance.AllSeq(contentSEORevisionGroup) {
		if err != nil {
			return nil, core.E("contentSEORevisionRecords", "iterate revisions", err)
		}

		revision, err := contentSEORevisionValue(entry.Value)
		if err != nil {
			return nil, err
		}
		if pageID != "" && revision.PageID != pageID {
			continue
		}
		if pendingOnly && revision.ScheduledAt != nil {
			continue
		}

		records = append(records, seoRevisionRecord{
			Key:      entry.Key,
			Revision: revision,
		})
	}

	return records, nil
}

var contentSEORevisionValue = func(value string) (SEORevision, error) {
	var revision SEORevision
	result := core.JSONUnmarshalString(value, &revision)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return SEORevision{}, core.E("contentSEORevisionValue", "decode revision", err)
		}
		return SEORevision{}, core.E("contentSEORevisionValue", "decode revision", nil)
	}
	if core.Trim(revision.PageID) == "" {
		return SEORevision{}, core.E("contentSEORevisionValue", "revision page_id is empty", nil)
	}
	return revision, nil
}

func contentSEORevisionKey(createdAt time.Time) string {
	return core.Concat(core.Sprint(createdAt.UnixNano()), "-", contentSEORandomHex())
}

func contentSEORandomHex() string {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "000000"
	}
	return hex.EncodeToString(bytes)
}

var contentSEOContextErr = func(operation string, ctx context.Context) error {
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	return core.E(operation, "cancelled", ctx.Err())
}

func contentSEOIsGooglebot(userAgent string) bool {
	return core.Contains(core.Lower(core.Trim(userAgent)), "googlebot")
}
