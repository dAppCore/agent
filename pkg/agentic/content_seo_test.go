// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleRevision_Good_CreatesPendingRevision(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	revision, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	require.NoError(t, err)
	assert.Equal(t, "/help/hosting", revision.PageID)
	assert.Equal(t, "Updated copy", revision.Content)
	assert.Nil(t, revision.ScheduledAt)
	assert.True(t, revision.CreatedAt.Equal(now))

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Nil(t, pending[0].ScheduledAt)

	var rawEntries []string
	subsystem.stateStoreRestore(contentSEORevisionGroup, func(_ string, value string) bool {
		rawEntries = append(rawEntries, value)
		return true
	})
	require.Len(t, rawEntries, 1)
	assert.Contains(t, rawEntries[0], `"scheduled_at":null`)
}

func TestScheduleRevision_Bad_EmptyPageID(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "", "Updated copy")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page_id is required")
}

func TestOnGooglebotVisit_Good_SetsPublishTimeInRange(t *testing.T) {
	withStateStoreTempDir(t)

	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, now)
	restoreContentSEORandomDelay(t, 37*time.Minute)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	_, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	require.NoError(t, err)
	require.NoError(t, subsystem.OnGooglebotVisit(context.Background(), "/help/hosting"))

	pending, err := subsystem.GetPendingRevisions("/help/hosting")
	require.NoError(t, err)
	assert.Len(t, pending, 0)

	records, err := subsystem.contentSEORevisionRecords(subsystem.stateStoreInstance(), "/help/hosting", false)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.NotNil(t, records[0].Revision.ScheduledAt)

	delta := records[0].Revision.ScheduledAt.Sub(now)
	assert.GreaterOrEqual(t, delta, 8*time.Minute)
	assert.LessOrEqual(t, delta, 62*time.Minute)
	assert.Equal(t, 37*time.Minute, delta)
}

func TestOnGooglebotVisit_Bad_NoPendingRevision(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	require.NoError(t, subsystem.OnGooglebotVisit(context.Background(), "/help/hosting"))
	assert.Equal(t, 0, subsystem.stateStoreCount(contentSEORevisionGroup))
}

func TestOnGooglebotVisit_Ugly_StoreError(t *testing.T) {
	root := t.TempDir()
	blocked := root + "/blocked"
	writeResult := fs.Write(blocked, "blocked")
	require.True(t, writeResult.OK)
	t.Setenv("CORE_WORKSPACE", blocked)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	err := subsystem.OnGooglebotVisit(context.Background(), "/help/hosting")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state store unavailable")
}

func TestContentSEO_RegisterTools_Good_RegistersScheduleTool(t *testing.T) {
	t.Setenv("CORE_MCP_FULL", "1")

	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	require.NoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	require.NoError(t, err)

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "content_seo_schedule")
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
