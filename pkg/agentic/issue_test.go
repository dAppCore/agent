// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestIssue_HandleIssueRecordCreate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "Fix auth", payload["title"])
		core.AssertEqual(t, "bug", payload["type"])
		core.AssertEqual(t, "codex", payload["assignee"])

		_, _ = w.Write([]byte(`{"data":{"slug":"fix-auth","title":"Fix auth","type":"bug","status":"open","priority":"high","assignee":"codex","labels":["auth"]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordCreate(context.Background(), core.NewOptions(
		core.Option{Key: "title", Value: "Fix auth"},
		core.Option{Key: "type", Value: "bug"},
		core.Option{Key: "priority", Value: "high"},
		core.Option{Key: "assignee", Value: "codex"},
		core.Option{Key: "labels", Value: "auth"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "fix-auth", output.Issue.Slug)
	core.AssertEqual(t, "open", output.Issue.Status)
	core.AssertEqual(t, "codex", output.Issue.Assignee)
	core.AssertEqual(t, []string{"auth"}, output.Issue.Labels)
}

func TestIssue_HandleIssueRecordGet_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleIssueRecordGet(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "issueGet: id or slug is required")
}

func TestIssue_HandleIssueRecordGet_Good_IDAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/42", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"id":42,"slug":"fix-auth","title":"Fix auth","status":"open"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordGet(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: "42"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 42, output.Issue.ID)
	core.AssertEqual(t, "fix-auth", output.Issue.Slug)
}

func TestIssue_HandleIssueRecordList_Good_Filters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues", r.URL.Path)
		core.AssertEqual(t, "open", r.URL.Query().Get("status"))
		core.AssertEqual(t, "bug", r.URL.Query().Get("type"))
		core.AssertEqual(t, "high", r.URL.Query().Get("priority"))
		core.AssertEqual(t, "codex", r.URL.Query().Get("assignee"))
		core.AssertEqual(t, []string{"auth", "backend"}, r.URL.Query()["labels"])
		_, _ = w.Write([]byte(`{"data":{"issues":[{"id":7,"workspace_id":3,"sprint_id":5,"slug":"fix-auth","title":"Fix auth","labels":["auth","backend"]}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordList(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "open"},
		core.Option{Key: "type", Value: "bug"},
		core.Option{Key: "priority", Value: "high"},
		core.Option{Key: "assignee", Value: "codex"},
		core.Option{Key: "labels", Value: []string{"auth", "backend"}},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueListOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Issues, 1)
	core.AssertEqual(t, 1, output.Count)
	core.AssertEqual(t, []string{"auth", "backend"}, output.Issues[0].Labels)
}

func TestIssue_HandleIssueRecordList_Bad_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"backend offline"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordList(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "open"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "issue.list")
}

func TestIssue_HandleIssueRecordAssign_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth", r.URL.Path)
		core.AssertEqual(t, http.MethodPatch, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "codex", payload["assignee"])

		_, _ = w.Write([]byte(`{"data":{"issue":{"slug":"fix-auth","title":"Fix auth","status":"assigned","assignee":"codex"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordAssign(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
		core.Option{Key: "assignee", Value: "codex"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "codex", output.Issue.Assignee)
	core.AssertEqual(t, "assigned", output.Issue.Status)
}

func TestIssue_HandleIssueRecordAssign_Bad_MissingAssignee(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleIssueRecordAssign(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "issueAssign: assignee is required")
}

func TestIssue_HandleIssueRecordAssign_Ugly_MissingIdentifier(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleIssueRecordAssign(context.Background(), core.NewOptions(
		core.Option{Key: "assignee", Value: "codex"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "issueAssign: id or slug is required")
}

func TestIssue_HandleIssueRecordReport_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth/comments", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "QA failed: build output changed", payload["body"])
		core.AssertEqual(t, "codex", payload["author"])

		_, _ = w.Write([]byte(`{"data":{"comment":{"id":88,"issue_id":42,"author":"codex","body":"QA failed: build output changed","metadata":{"source":"qa"}}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordReport(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
		core.Option{Key: "report", Value: "QA failed: build output changed"},
		core.Option{Key: "author", Value: "codex"},
		core.Option{Key: "metadata", Value: map[string]any{"source": "qa"}},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueReportOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 88, output.Comment.ID)
	core.AssertEqual(t, "QA failed: build output changed", output.Comment.Body)
	core.AssertEqual(t, "codex", output.Comment.Author)
	core.AssertEqual(t, map[string]any{"source": "qa"}, output.Comment.Metadata)
}

func TestIssue_HandleIssueRecordReport_Bad_MissingReport(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleIssueRecordReport(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "issueReport: report is required")
}

func TestIssue_HandleIssueRecordReport_Ugly_MissingIdentifier(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleIssueRecordReport(context.Background(), core.NewOptions(
		core.Option{Key: "report", Value: "QA failed: build output changed"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "issueReport: issue_id, id, or slug is required")
}

func TestIssue_HandleIssueRecordList_Ugly_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues", r.URL.Path)
		core.AssertEqual(t, "open", r.URL.Query().Get("status"))
		_, _ = w.Write([]byte(`{"data":{"issues":[{"id":7,"workspace_id":3,"sprint_id":5,"slug":"fix-auth","title":"Fix auth","labels":["auth","backend"]}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordList(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "open"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueListOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Issues, 1)
	core.AssertEqual(t, 1, output.Count)
	core.AssertEqual(t, 3, output.Issues[0].WorkspaceID)
	core.AssertEqual(t, 5, output.Issues[0].SprintID)
	core.AssertEqual(t, []string{"auth", "backend"}, output.Issues[0].Labels)
}
