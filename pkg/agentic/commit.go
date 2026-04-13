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

	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("commitWorkspace", "status not found", nil)
		}
		return CommitOutput{}, err
	}

	metaDir := WorkspaceMetaDir(workspaceDir)
	if r := fs.EnsureDir(metaDir); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("commitWorkspace", "failed to create metadata directory", nil)
		}
		return CommitOutput{}, err
	}

	journalPath := core.JoinPath(metaDir, "journal.jsonl")
	markerPath := core.JoinPath(metaDir, "commit.json")

	committedAt := time.Now().UTC().Format(time.RFC3339)
	if existingCommit, ok := readCommitMarker(markerPath); ok && existingCommit.UpdatedAt == workspaceStatus.UpdatedAt && existingCommit.Runs == workspaceStatus.Runs {
		return CommitOutput{
			Success:     true,
			Workspace:   input.Workspace,
			JournalPath: journalPath,
			MarkerPath:  markerPath,
			CommittedAt: existingCommit.CommittedAt,
			Skipped:     true,
		}, nil
	}

	record := commitWorkspaceRecord(workspaceDir, workspaceStatus, committedAt)
	line := core.Concat(core.JSONMarshalString(record), "\n")

	appendHandle := fs.Append(journalPath)
	if !appendHandle.OK {
		err, _ := appendHandle.Value.(error)
		if err == nil {
			err = core.E("commitWorkspace", "failed to open journal", nil)
		}
		return CommitOutput{}, err
	}
	core.WriteAll(appendHandle.Value, line)

	marker := commitMarker{
		Workspace:   WorkspaceName(workspaceDir),
		UpdatedAt:   workspaceStatus.UpdatedAt,
		Runs:        workspaceStatus.Runs,
		CommittedAt: committedAt,
	}
	if r := fs.WriteAtomic(markerPath, core.JSONMarshalString(marker)); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("commitWorkspace", "failed to write commit marker", nil)
		}
		return CommitOutput{}, err
	}

	return CommitOutput{
		Success:     true,
		Workspace:   input.Workspace,
		JournalPath: journalPath,
		MarkerPath:  markerPath,
		CommittedAt: committedAt,
	}, nil
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
