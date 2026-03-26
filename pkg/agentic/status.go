// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"syscall"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Workspace status file convention:
//
//   {workspace}/status.json  — current state of the workspace
//   {workspace}/BLOCKED.md   — question the agent needs answered (written by agent)
//   {workspace}/ANSWER.md    — response from human (written by reviewer)
//
// Status lifecycle:
//   running → completed     (normal finish)
//   running → blocked       (agent wrote BLOCKED.md and exited)
//   blocked → running       (resume after ANSWER.md provided)
//   completed → merged      (PR verified and auto-merged)
//   running → failed        (agent crashed / non-zero exit)

// WorkspaceStatus represents the current state of an agent workspace.
//
//	st, err := ReadStatus(wsDir)
//	if err == nil && st.Status == "completed" { autoCreatePR(wsDir) }
type WorkspaceStatus struct {
	Status    string    `json:"status"`             // running, completed, blocked, failed
	Agent     string    `json:"agent"`              // gemini, claude, codex
	Repo      string    `json:"repo"`               // target repo
	Org       string    `json:"org,omitempty"`      // forge org (e.g. "core")
	Task      string    `json:"task"`               // task description
	Branch    string    `json:"branch,omitempty"`   // git branch name
	Issue     int       `json:"issue,omitempty"`      // forge issue number
	PID       int       `json:"pid,omitempty"`        // OS process ID (if running)
	ProcessID string    `json:"process_id,omitempty"` // go-process ID for managed lookup
	StartedAt time.Time `json:"started_at"`           // when dispatch started
	UpdatedAt time.Time `json:"updated_at"`         // last status change
	Question  string    `json:"question,omitempty"` // from BLOCKED.md
	Runs      int       `json:"runs"`               // how many times dispatched/resumed
	PRURL     string    `json:"pr_url,omitempty"`   // pull request URL (after PR created)
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
	status.UpdatedAt = time.Now()
	statusPath := core.JoinPath(wsDir, "status.json")
	if r := fs.WriteAtomic(statusPath, core.JSONMarshalString(status)); !r.OK {
		err, _ := r.Value.(error)
		return core.E("writeStatus", "failed to write status", err)
	}
	return nil
}

// ReadStatus parses the status.json in a workspace directory.
//
//	st, err := agentic.ReadStatus("/path/to/workspace")
func ReadStatus(wsDir string) (*WorkspaceStatus, error) {
	r := fs.Read(core.JoinPath(wsDir, "status.json"))
	if !r.OK {
		return nil, core.E("ReadStatus", "status not found", nil)
	}
	var s WorkspaceStatus
	if ur := core.JSONUnmarshalString(r.Value.(string), &s); !ur.OK {
		err, _ := ur.Value.(error)
		return nil, core.E("ReadStatus", "invalid status json", err)
	}
	return &s, nil
}

// --- agentic_status tool ---

// StatusInput is the input for agentic_status.
//
//	input := agentic.StatusInput{Workspace: "go-io-123", Limit: 50}
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
	Total     int             `json:"total"`
	Running   int             `json:"running"`
	Queued    int             `json:"queued"`
	Completed int             `json:"completed"`
	Failed    int             `json:"failed"`
	Blocked   []BlockedInfo   `json:"blocked,omitempty"`
}

// BlockedInfo shows a workspace that needs human input.
//
//	info := agentic.BlockedInfo{Name: "go-io/task-4", Repo: "go-io", Question: "Which API version?"}
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
	wsRoot := WorkspaceRoot()

	// Scan both old (*/status.json) and new (*/*/*/status.json) layouts
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))
	statusFiles := append(old, deep...)

	var out StatusOutput

	for _, statusPath := range statusFiles {
		wsDir := core.PathDir(statusPath)
		name := wsDir[len(wsRoot)+1:]

		st, err := ReadStatus(wsDir)
		if err != nil {
			out.Total++
			out.Failed++
			continue
		}

		// If status is "running", check if PID is still alive
		if st.Status == "running" && st.PID > 0 {
			if err := syscall.Kill(st.PID, 0); err != nil {
				blockedPath := core.JoinPath(wsDir, "repo", "BLOCKED.md")
				if r := fs.Read(blockedPath); r.OK {
					st.Status = "blocked"
					st.Question = core.Trim(r.Value.(string))
				} else {
					logFile := core.JoinPath(wsDir, core.Sprintf("agent-%s.log", st.Agent))
					if r := fs.Read(logFile); !r.OK {
						st.Status = "failed"
						st.Question = "Agent process died (no output log)"
					} else {
						st.Status = "completed"
					}
				}
				writeStatus(wsDir, st)
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
