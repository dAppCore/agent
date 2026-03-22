// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
//	st, err := readStatus(wsDir)
//	if err == nil && st.Status == "completed" { autoCreatePR(wsDir) }
type WorkspaceStatus struct {
	Status    string    `json:"status"`             // running, completed, blocked, failed
	Agent     string    `json:"agent"`              // gemini, claude, codex
	Repo      string    `json:"repo"`               // target repo
	Org       string    `json:"org,omitempty"`      // forge org (e.g. "core")
	Task      string    `json:"task"`               // task description
	Branch    string    `json:"branch,omitempty"`   // git branch name
	Issue     int       `json:"issue,omitempty"`    // forge issue number
	PID       int       `json:"pid,omitempty"`      // process ID (if running)
	StartedAt time.Time `json:"started_at"`         // when dispatch started
	UpdatedAt time.Time `json:"updated_at"`         // last status change
	Question  string    `json:"question,omitempty"` // from BLOCKED.md
	Runs      int       `json:"runs"`               // how many times dispatched/resumed
	PRURL     string    `json:"pr_url,omitempty"`   // pull request URL (after PR created)
}

func writeStatus(wsDir string, status *WorkspaceStatus) error {
	status.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	if r := fs.Write(core.JoinPath(wsDir, "status.json"), string(data)); !r.OK {
		err, _ := r.Value.(error)
		return core.E("writeStatus", "failed to write status", err)
	}
	return nil
}

func readStatus(wsDir string) (*WorkspaceStatus, error) {
	r := fs.Read(core.JoinPath(wsDir, "status.json"))
	if !r.OK {
		return nil, core.E("readStatus", "status not found", nil)
	}
	var s WorkspaceStatus
	if err := json.Unmarshal([]byte(r.Value.(string)), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- agentic_status tool ---

// StatusInput is the input for agentic_status.
//
//	input := agentic.StatusInput{Workspace: "go-io-123"}
type StatusInput struct {
	Workspace string `json:"workspace,omitempty"` // specific workspace name, or empty for all
}

// StatusOutput is the output for agentic_status.
//
//	out := agentic.StatusOutput{Count: 1, Workspaces: []agentic.WorkspaceInfo{{Name: "go-io-123"}}}
type StatusOutput struct {
	Workspaces []WorkspaceInfo `json:"workspaces"`
	Count      int             `json:"count"`
}

// WorkspaceInfo summarises one workspace returned by agentic_status.
//
//	info := agentic.WorkspaceInfo{Name: "go-io-123", Status: "running", Agent: "codex", Repo: "go-io"}
type WorkspaceInfo struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Agent    string `json:"agent"`
	Repo     string `json:"repo"`
	Task     string `json:"task"`
	Age      string `json:"age"`
	Question string `json:"question,omitempty"`
	Runs     int    `json:"runs"`
}

func (s *PrepSubsystem) registerStatusTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_status",
		Description: "List agent workspaces and their status (running, completed, blocked, failed). Shows blocked agents with their questions.",
	}, s.status)
}

func (s *PrepSubsystem) status(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	wsRoot := WorkspaceRoot()

	r := fs.List(wsRoot)
	if !r.OK {
		return nil, StatusOutput{}, core.E("status", "no workspaces found", nil)
	}
	entries := r.Value.([]os.DirEntry)

	var workspaces []WorkspaceInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Filter by specific workspace if requested
		if input.Workspace != "" && name != input.Workspace {
			continue
		}

		wsDir := core.JoinPath(wsRoot, name)
		info := WorkspaceInfo{Name: name}

		// Try reading status.json
		st, err := readStatus(wsDir)
		if err != nil {
			// Legacy workspace (no status.json) — check for log file
			logFiles, _ := filepath.Glob(core.JoinPath(wsDir, "agent-*.log"))
			if len(logFiles) > 0 {
				info.Status = "completed"
			} else {
				info.Status = "unknown"
			}
			fi, _ := entry.Info()
			if fi != nil {
				info.Age = time.Since(fi.ModTime()).Truncate(time.Minute).String()
			}
			workspaces = append(workspaces, info)
			continue
		}

		info.Status = st.Status
		info.Agent = st.Agent
		info.Repo = st.Repo
		info.Task = st.Task
		info.Runs = st.Runs
		info.Age = time.Since(st.StartedAt).Truncate(time.Minute).String()

		// If status is "running", check if PID is still alive
		if st.Status == "running" && st.PID > 0 {
			if err := syscall.Kill(st.PID, 0); err != nil {
				// Process died — check for BLOCKED.md
				blockedPath := core.JoinPath(wsDir, "repo", "BLOCKED.md")
				if r := fs.Read(blockedPath); r.OK {
					info.Status = "blocked"
					info.Question = core.Trim(r.Value.(string))
					st.Status = "blocked"
					st.Question = info.Question
				} else {
					// Dead PID without BLOCKED.md — check exit code from log
					// If no evidence of success, mark as failed (not completed)
					logFile := core.JoinPath(wsDir, core.Sprintf("agent-%s.log", st.Agent))
					if r := fs.Read(logFile); !r.OK {
						info.Status = "failed"
						st.Status = "failed"
						st.Question = "Agent process died (no output log)"
					} else {
						info.Status = "completed"
						st.Status = "completed"
					}
				}
				writeStatus(wsDir, st)
			}
		}

		if st.Status == "blocked" {
			info.Question = st.Question
		}

		workspaces = append(workspaces, info)
	}

	return nil, StatusOutput{
		Workspaces: workspaces,
		Count:      len(workspaces),
	}, nil
}
