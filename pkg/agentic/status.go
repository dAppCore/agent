// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Workspace status file convention:
//
//   {workspace}/status.json  — current state of the workspace
//   {workspace}/repo/BLOCKED.md   — question the agent needs answered (written by agent)
//   {workspace}/repo/ANSWER.md    — response from human (written by reviewer)
//   {workspace}/.meta/agent-*.log — captured agent output
//
// Status lifecycle:
//   running → completed     (normal finish)
//   running → blocked       (agent wrote BLOCKED.md and exited)
//   blocked → running       (resume after ANSWER.md provided)
//   completed → merged      (PR verified and auto-merged)
//   running → failed        (agent crashed / non-zero exit)

// WorkspaceStatus represents the current state of an agent workspace.
//
//	r := ReadStatusResult(wsDir)
//	if r.OK && r.Value.(*WorkspaceStatus).Status == "completed" { autoCreatePR(wsDir) }
type WorkspaceStatus struct {
	Status    string    `json:"status"`               // running, completed, blocked, failed
	Agent     string    `json:"agent"`                // gemini, claude, codex
	Repo      string    `json:"repo"`                 // target repo
	Org       string    `json:"org,omitempty"`        // forge org (e.g. "core")
	Task      string    `json:"task"`                 // task description
	Branch    string    `json:"branch,omitempty"`     // git branch name
	Issue     int       `json:"issue,omitempty"`      // forge issue number
	PID       int       `json:"pid,omitempty"`        // OS process ID (if running)
	ProcessID string    `json:"process_id,omitempty"` // go-process ID for managed lookup
	StartedAt time.Time `json:"started_at"`           // when dispatch started
	UpdatedAt time.Time `json:"updated_at"`           // last status change
	Question  string    `json:"question,omitempty"`   // from BLOCKED.md
	Runs      int       `json:"runs"`                 // how many times dispatched/resumed
	PRURL     string    `json:"pr_url,omitempty"`     // pull request URL (after PR created)
}

// WorkspaceQuery is the QUERY type for workspace state lookups.
// Returns the workspace Registry via c.QUERY(agentic.WorkspaceQuery{}).
//
//	r := c.QUERY(agentic.WorkspaceQuery{})
//	if r.OK { reg := r.Value.(*core.Registry[*WorkspaceStatus]) }
//	r := c.QUERY(agentic.WorkspaceQuery{Name: "core/go-io/task-5"})
type WorkspaceQuery struct {
	Name   string // specific workspace (empty = all)
	Status string // filter by status (empty = all)
}

func writeStatus(wsDir string, status *WorkspaceStatus) error {
	r := writeStatusResult(wsDir, status)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("writeStatus", "failed to write status", nil)
		}
		return err
	}
	return nil
}

// writeStatusResult writes status.json and returns core.Result.
//
//	r := writeStatusResult("/srv/core/workspace/core/go-io/task-5", &WorkspaceStatus{Status: "running"})
//	if r.OK { return }
func writeStatusResult(wsDir string, status *WorkspaceStatus) core.Result {
	if status == nil {
		return core.Result{Value: core.E("writeStatus", "status is required", nil), OK: false}
	}
	status.UpdatedAt = time.Now()
	statusPath := WorkspaceStatusPath(wsDir)
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

// ReadStatus parses the status.json in a workspace directory.
//
// Deprecated: use ReadStatusResult.
//
//	st, err := agentic.ReadStatus("/path/to/workspace")
func ReadStatus(wsDir string) (*WorkspaceStatus, error) {
	r := ReadStatusResult(wsDir)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return nil, core.E("ReadStatus", "failed to read status", nil)
		}
		return nil, err
	}

	st, ok := r.Value.(*WorkspaceStatus)
	if !ok || st == nil {
		return nil, core.E("ReadStatus", "invalid status payload", nil)
	}
	return st, nil
}

// ReadStatusResult parses status.json and returns a WorkspaceStatus pointer.
//
//	r := ReadStatusResult("/path/to/workspace")
//	if r.OK { st := r.Value.(*WorkspaceStatus) }
func ReadStatusResult(wsDir string) core.Result {
	r := fs.Read(WorkspaceStatusPath(wsDir))
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("ReadStatusResult", "status not found", nil), OK: false}
		}
		return core.Result{Value: core.E("ReadStatusResult", core.Concat("status not found for ", wsDir), err), OK: false}
	}
	var s WorkspaceStatus
	if ur := core.JSONUnmarshalString(r.Value.(string), &s); !ur.OK {
		err, _ := ur.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("ReadStatusResult", "invalid status json", nil), OK: false}
		}
		return core.Result{Value: core.E("ReadStatusResult", "invalid status json", err), OK: false}
	}
	return core.Result{Value: &s, OK: true}
}

// workspaceStatusValue extracts a WorkspaceStatus from a Result.
//
//	r := ReadStatusResult("/path/to/workspace")
//	st, ok := workspaceStatusValue(r)
func workspaceStatusValue(result core.Result) (*WorkspaceStatus, bool) {
	st, ok := result.Value.(*WorkspaceStatus)
	if !ok || st == nil {
		return nil, false
	}
	return st, true
}

// --- agentic_status tool ---

// StatusInput is the input for agentic_status.
//
//	input := agentic.StatusInput{Workspace: "core/go-io/task-42", Limit: 50}
type StatusInput struct {
	Workspace string `json:"workspace,omitempty"` // specific workspace name, or empty for all
	Limit     int    `json:"limit,omitempty"`     // max results (default 100)
	Status    string `json:"status,omitempty"`    // filter: running, completed, failed, blocked
}

// StatusOutput is the output for agentic_status.
// Returns stats by default. Only blocked workspaces are listed (they need attention).
//
//	out := agentic.StatusOutput{Total: 42, Running: 3, Queued: 10, Completed: 25}
type StatusOutput struct {
	Total     int           `json:"total"`
	Running   int           `json:"running"`
	Queued    int           `json:"queued"`
	Completed int           `json:"completed"`
	Failed    int           `json:"failed"`
	Blocked   []BlockedInfo `json:"blocked,omitempty"`
}

// BlockedInfo shows a workspace that needs human input.
//
//	info := agentic.BlockedInfo{Name: "core/go-io/task-4", Repo: "go-io", Question: "Which API version?"}
type BlockedInfo struct {
	Name     string `json:"name"`
	Repo     string `json:"repo"`
	Agent    string `json:"agent"`
	Question string `json:"question"`
}

func (s *PrepSubsystem) registerStatusTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_status",
		Description: "List agent workspaces and their status (running, completed, blocked, failed). Shows blocked agents with their questions.",
	}, s.status)
}

func (s *PrepSubsystem) status(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	statusFiles := WorkspaceStatusPaths()
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}

	var out StatusOutput

	for _, statusPath := range statusFiles {
		wsDir := core.PathDir(statusPath)
		name := WorkspaceName(wsDir)

		result := ReadStatusResult(wsDir)
		st, ok := workspaceStatusValue(result)
		if !ok {
			out.Total++
			out.Failed++
			continue
		}

		// If status is "running", check whether the managed process is still alive.
		if st.Status == "running" && (st.ProcessID != "" || st.PID > 0) {
			if !ProcessAlive(runtime, st.ProcessID, st.PID) {
				blockedPath := workspaceBlockedPath(wsDir)
				if r := fs.Read(blockedPath); r.OK {
					st.Status = "blocked"
					st.Question = core.Trim(r.Value.(string))
				} else {
					if len(workspaceLogFiles(wsDir)) == 0 {
						st.Status = "failed"
						st.Question = "Agent process died (no output log)"
					} else {
						st.Status = "completed"
					}
				}
				writeStatusResult(wsDir, st)
			}
		}

		out.Total++
		switch st.Status {
		case "running":
			out.Running++
		case "queued":
			out.Queued++
		case "completed":
			out.Completed++
		case "failed":
			out.Failed++
		case "blocked":
			out.Blocked = append(out.Blocked, BlockedInfo{
				Name:     name,
				Repo:     st.Repo,
				Agent:    st.Agent,
				Question: st.Question,
			})
		}
	}

	return nil, out, nil
}
