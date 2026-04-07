// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// result := ReadStatusResult(workspaceDir)
// if result.OK && result.Value.(*WorkspaceStatus).Status == "completed" { autoCreatePR(workspaceDir) }
type WorkspaceStatus struct {
	Status    string    `json:"status"`
	Agent     string    `json:"agent"`
	Repo      string    `json:"repo"`
	Org       string    `json:"org,omitempty"`
	Task      string    `json:"task"`
	Branch    string    `json:"branch,omitempty"`
	Issue     int       `json:"issue,omitempty"`
	PID       int       `json:"pid,omitempty"`
	ProcessID string    `json:"process_id,omitempty"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Question  string    `json:"question,omitempty"`
	Runs      int       `json:"runs"`
	PRURL     string    `json:"pr_url,omitempty"`
}

// r := c.QUERY(agentic.WorkspaceQuery{})
// if r.OK { reg := r.Value.(*core.Registry[*WorkspaceStatus]) }
// r := c.QUERY(agentic.WorkspaceQuery{Name: "core/go-io/task-5"})
type WorkspaceQuery struct {
	Name   string
	Status string
}

func writeStatus(workspaceDir string, status *WorkspaceStatus) error {
	r := writeStatusResult(workspaceDir, status)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("writeStatus", "failed to write status", nil)
		}
		return err
	}
	return nil
}

// result := writeStatusResult("/srv/core/workspace/core/go-io/task-5", &WorkspaceStatus{Status: "running"})
// if result.OK { return }
func writeStatusResult(workspaceDir string, status *WorkspaceStatus) core.Result {
	if status == nil {
		return core.Result{Value: core.E("writeStatus", "status is required", nil), OK: false}
	}
	status.UpdatedAt = time.Now()
	statusPath := WorkspaceStatusPath(workspaceDir)
	if r := fs.WriteAtomic(statusPath, core.JSONMarshalString(status)); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			core.Warn("agentic.writeStatus: failed to write status", "path", statusPath)
			return core.Result{Value: core.E("writeStatus", "failed to write status", nil), OK: false}
		}
		core.Warn("agentic.writeStatus: failed to write status", "path", statusPath, "reason", err)
		return core.Result{Value: core.E("writeStatus", "failed to write status", err), OK: false}
	}
	return core.Result{OK: true}
}

// result := ReadStatusResult("/path/to/workspace")
// if result.OK { workspaceStatus := result.Value.(*WorkspaceStatus) }
func ReadStatusResult(workspaceDir string) core.Result {
	r := fs.Read(WorkspaceStatusPath(workspaceDir))
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("ReadStatusResult", "status not found", nil), OK: false}
		}
		return core.Result{Value: core.E("ReadStatusResult", core.Concat("status not found for ", workspaceDir), err), OK: false}
	}
	var workspaceStatus WorkspaceStatus
	if parseResult := core.JSONUnmarshalString(r.Value.(string), &workspaceStatus); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("ReadStatusResult", "invalid status json", nil), OK: false}
		}
		return core.Result{Value: core.E("ReadStatusResult", "invalid status json", err), OK: false}
	}
	return core.Result{Value: &workspaceStatus, OK: true}
}

// read, err := ReadStatus("/path/to/workspace")
// if err == nil { core.Println(read.Status) }
func ReadStatus(workspaceDir string) (*WorkspaceStatus, error) {
	result := ReadStatusResult(workspaceDir)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("ReadStatus", "failed to read status", nil)
		}
		return nil, err
	}

	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		return nil, core.E("ReadStatus", "invalid status payload", nil)
	}
	return workspaceStatus, nil
}

// result := ReadStatusResult("/path/to/workspace")
// workspaceStatus, ok := workspaceStatusValue(result)
func workspaceStatusValue(result core.Result) (*WorkspaceStatus, bool) {
	workspaceStatus, ok := result.Value.(*WorkspaceStatus)
	if !ok || workspaceStatus == nil {
		return nil, false
	}
	return workspaceStatus, true
}

// input := agentic.StatusInput{Workspace: "core/go-io/task-42", Status: "blocked", Limit: 50}
type StatusInput struct {
	Workspace string `json:"workspace,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Status    string `json:"status,omitempty"`
}

// out := agentic.StatusOutput{Total: 42, Running: 3, Queued: 10, Completed: 25}
type StatusOutput struct {
	Total     int           `json:"total"`
	Running   int           `json:"running"`
	Queued    int           `json:"queued"`
	Completed int           `json:"completed"`
	Failed    int           `json:"failed"`
	Blocked   []BlockedInfo `json:"blocked,omitempty"`
}

// info := agentic.BlockedInfo{Name: "core/go-io/task-4", Repo: "go-io", Question: "Which API version?"}
type BlockedInfo struct {
	Name     string `json:"name"`
	Repo     string `json:"repo"`
	Agent    string `json:"agent"`
	Question string `json:"question"`
}

func (s *PrepSubsystem) registerStatusTool(svc *coremcp.Service) {
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_status",
		Description: "List agent workspaces and their status (running, completed, blocked, failed). Supports workspace, status, and limit filters. Shows blocked agents with their questions.",
	}, s.status)
}

func (s *PrepSubsystem) status(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	statusFiles := WorkspaceStatusPaths()
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}

	var statusSummary StatusOutput
	matched := 0

	for _, statusPath := range statusFiles {
		workspaceDir := core.PathDir(statusPath)
		name := WorkspaceName(workspaceDir)
		if !statusInputMatchesWorkspace(input.Workspace, workspaceDir, name) {
			continue
		}

		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			if input.Status != "" && input.Status != "failed" {
				continue
			}
			if !statusInputMatchesStatus(input.Status, "failed") {
				continue
			}
			statusSummary.Total++
			statusSummary.Failed++
			matched++
			if input.Limit > 0 && matched >= input.Limit {
				break
			}
			continue
		}

		if workspaceStatus.Status == "running" && (workspaceStatus.ProcessID != "" || workspaceStatus.PID > 0) {
			if !ProcessAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID) {
				blockedPath := workspaceBlockedPath(workspaceDir)
				if r := fs.Read(blockedPath); r.OK {
					workspaceStatus.Status = "blocked"
					workspaceStatus.Question = core.Trim(r.Value.(string))
				} else {
					if len(workspaceLogFiles(workspaceDir)) == 0 {
						workspaceStatus.Status = "failed"
						workspaceStatus.Question = "Agent process died (no output log)"
					} else {
						workspaceStatus.Status = "completed"
					}
				}
				writeStatusResult(workspaceDir, workspaceStatus)
			}
		}

		if !statusInputMatchesStatus(input.Status, workspaceStatus.Status) {
			continue
		}

		statusSummary.Total++
		switch workspaceStatus.Status {
		case "running":
			statusSummary.Running++
		case "queued":
			statusSummary.Queued++
		case "completed":
			statusSummary.Completed++
		case "failed":
			statusSummary.Failed++
		case "blocked":
			statusSummary.Blocked = append(statusSummary.Blocked, BlockedInfo{
				Name:     name,
				Repo:     workspaceStatus.Repo,
				Agent:    workspaceStatus.Agent,
				Question: workspaceStatus.Question,
			})
		}
		matched++
		if input.Limit > 0 && matched >= input.Limit {
			break
		}
	}

	return nil, statusSummary, nil
}

func statusInputMatchesWorkspace(requested, workspaceDir, workspaceName string) bool {
	if requested == "" {
		return true
	}
	if requested == workspaceName {
		return true
	}
	if requested == workspaceDir {
		return true
	}
	return false
}

func statusInputMatchesStatus(requested, current string) bool {
	if requested == "" {
		return true
	}
	return requested == current
}
