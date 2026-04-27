// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.CommitInput{Workspace: "core/go-io/task-42"}
type CommitInput struct {
	Workspace string `json:"workspace"`
}

// out := agentic.CommitOutput{Success: true, Workspace: "core/go-io/task-42", JournalPath: "/srv/.core/workspace/core/go-io/task-42/.meta/journal.jsonl"}
type CommitOutput struct {
	Success     bool   `json:"success"`
	Workspace   string `json:"workspace"`
	JournalPath string `json:"journal_path,omitempty"`
	MarkerPath  string `json:"marker_path,omitempty"`
	CommittedAt string `json:"committed_at,omitempty"`
	Skipped     bool   `json:"skipped,omitempty"`
}

// result := c.Action("agentic.commit").Run(ctx, core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-42"}))
func (s *PrepSubsystem) handleCommit(_ context.Context, options core.Options) core.Result {
	input := CommitInput{
		Workspace: optionStringValue(options, "workspace"),
	}
	output, err := s.commitWorkspace(nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerCommitTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_commit",
		Description: "Write the final workspace dispatch record to the local journal after verify completes.",
	}, s.commitTool)
}

func (s *PrepSubsystem) commitTool(ctx context.Context, _ *mcp.CallToolRequest, input CommitInput) (*mcp.CallToolResult, CommitOutput, error) {
	output, err := s.commitWorkspace(ctx, input)
	if err != nil {
		return nil, CommitOutput{}, err
	}
	return nil, output, nil
}

func (s *PrepSubsystem) commitWorkspace(ctx context.Context, input CommitInput) (CommitOutput, error) {
	workspaceDir := resolveWorkspace(input.Workspace)
	if workspaceDir == "" {
		return CommitOutput{}, core.E("commitWorkspace", core.Concat("workspace not found: ", input.Workspace), nil)
	}

	workspaceStatus, err := commitWorkspaceStatus(workspaceDir)
	if err != nil {
		return CommitOutput{}, err
	}

	metaDir := WorkspaceMetaDir(workspaceDir)
	if err := commitEnsureMetaDir(metaDir); err != nil {
		return CommitOutput{}, err
	}

	journalPath := core.JoinPath(metaDir, "journal.jsonl")
	markerPath := core.JoinPath(metaDir, "commit.json")

	committedAt := time.Now().UTC().Format(time.RFC3339)
	if existingCommit, ok := readCommitMarker(markerPath); ok && existingCommit.UpdatedAt == workspaceStatus.UpdatedAt && existingCommit.Runs == workspaceStatus.Runs {
		return commitSkippedOutput(input.Workspace, journalPath, markerPath, existingCommit), nil
	}

	record := commitWorkspaceRecord(workspaceDir, workspaceStatus, committedAt)
	if err := commitAppendJournal(journalPath, record); err != nil {
		return CommitOutput{}, err
	}

	if err := commitWriteMarker(markerPath, workspaceDir, workspaceStatus, committedAt); err != nil {
		return CommitOutput{}, err
	}

	// Mirror the dispatch record to the top-level dispatch_history group so
	// sync push can drain completed dispatches without re-scanning the
	// workspace tree — RFC §15.5 + §16.3. The record carries the same
	// shape expected by `POST /v1/agent/sync`.
	record["id"] = WorkspaceName(workspaceDir)
	record["synced"] = false
	s.stateStoreSet(stateDispatchHistoryGroup, WorkspaceName(workspaceDir), record)

	// RFC §15.5 — write the permanent stats row to `.core/workspace/db.duckdb`
	// so the "what happened in the last 50 dispatches" query answer survives
	// even after `dispatch_history` drains to the platform.
	s.recordWorkspaceStats(workspaceDir, workspaceStatus)

	return CommitOutput{
		Success:     true,
		Workspace:   input.Workspace,
		JournalPath: journalPath,
		MarkerPath:  markerPath,
		CommittedAt: committedAt,
	}, nil
}

func commitWorkspaceStatus(workspaceDir string) (*WorkspaceStatus, error) {
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if ok {
		return workspaceStatus, nil
	}
	err, _ := result.Value.(error)
	if err == nil {
		err = core.E("commitWorkspace", "status not found", nil)
	}
	return nil, err
}

func commitEnsureMetaDir(metaDir string) error {
	if r := fs.EnsureDir(metaDir); r.OK {
		return nil
	}
	err, _ := r.Value.(error)
	if err == nil {
		err = core.E("commitWorkspace", "failed to create metadata directory", nil)
	}
	return err
}

func commitSkippedOutput(workspace, journalPath, markerPath string, existingCommit commitMarker) CommitOutput {
	return CommitOutput{
		Success:     true,
		Workspace:   workspace,
		JournalPath: journalPath,
		MarkerPath:  markerPath,
		CommittedAt: existingCommit.CommittedAt,
		Skipped:     true,
	}
}

func commitAppendJournal(journalPath string, record map[string]any) error {
	appendHandle := fs.Append(journalPath)
	if !appendHandle.OK {
		err, _ := appendHandle.Value.(error)
		if err == nil {
			err = core.E("commitWorkspace", "failed to open journal", nil)
		}
		return err
	}

	line := core.Concat(core.JSONMarshalString(record), "\n")
	writeResult := core.WriteAll(appendHandle.Value, line)
	if writeResult.OK {
		return nil
	}

	err, _ := writeResult.Value.(error)
	if err == nil {
		err = core.E("commitWorkspace", "failed to append journal entry", nil)
	}
	return err
}

func commitWriteMarker(markerPath, workspaceDir string, workspaceStatus *WorkspaceStatus, committedAt string) error {
	marker := commitMarker{
		Workspace:   WorkspaceName(workspaceDir),
		UpdatedAt:   workspaceStatus.UpdatedAt,
		Runs:        workspaceStatus.Runs,
		CommittedAt: committedAt,
	}

	if r := fs.WriteAtomic(markerPath, core.JSONMarshalString(marker)); r.OK {
		return nil
	}

	err, _ := r.Value.(error)
	if err == nil {
		err = core.E("commitWorkspace", "failed to write commit marker", nil)
	}
	return err
}

type commitMarker struct {
	Workspace   string    `json:"workspace"`
	UpdatedAt   time.Time `json:"updated_at"`
	Runs        int       `json:"runs"`
	CommittedAt string    `json:"committed_at"`
}

func readCommitMarker(markerPath string) (commitMarker, bool) {
	r := fs.Read(markerPath)
	if !r.OK {
		return commitMarker{}, false
	}

	var marker commitMarker
	if parseResult := core.JSONUnmarshalString(r.Value.(string), &marker); !parseResult.OK {
		backupPath := core.Concat(markerPath, ".corrupt-", time.Now().UTC().Format("20060102T150405Z"))
		core.Warn("agentic.commit: corrupt commit marker", "path", markerPath, "backup", backupPath, "reason", parseResult.Value)
		if renameResult := fs.Rename(markerPath, backupPath); !renameResult.OK {
			core.Warn("agentic.commit: failed to preserve corrupt commit marker", "path", markerPath, "backup", backupPath, "reason", renameResult.Value)
		}
		return commitMarker{}, false
	}
	return marker, true
}

func commitWorkspaceRecord(workspaceDir string, workspaceStatus *WorkspaceStatus, committedAt string) map[string]any {
	record := map[string]any{
		"workspace":    WorkspaceName(workspaceDir),
		"repo":         workspaceStatus.Repo,
		"org":          workspaceStatus.Org,
		"task":         workspaceStatus.Task,
		"agent":        workspaceStatus.Agent,
		"branch":       workspaceStatus.Branch,
		"status":       workspaceStatus.Status,
		"question":     workspaceStatus.Question,
		"issue":        workspaceStatus.Issue,
		"runs":         workspaceStatus.Runs,
		"process_id":   workspaceStatus.ProcessID,
		"pr_url":       workspaceStatus.PRURL,
		"started_at":   workspaceStatus.StartedAt,
		"updated_at":   workspaceStatus.UpdatedAt,
		"committed_at": committedAt,
	}

	if report := readSyncWorkspaceReport(workspaceDir); len(report) > 0 {
		record["report"] = report
		if findings := anyMapSliceValue(report["findings"]); len(findings) > 0 {
			record["findings"] = findings
		}
		if changes := anyMapValue(report["changes"]); len(changes) > 0 {
			record["changes"] = changes
		}
	}

	return record
}
