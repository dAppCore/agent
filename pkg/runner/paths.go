// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
)

// fs reuses the shared unrestricted filesystem used by agentic.
var fs = agentic.LocalFs()

func runnerWorkspaceStatusFromAgentic(status *agentic.WorkspaceStatus) *WorkspaceStatus {
	if status == nil {
		return nil
	}
	return &WorkspaceStatus{
		Status:    status.Status,
		Agent:     status.Agent,
		Repo:      status.Repo,
		Org:       status.Org,
		Task:      status.Task,
		Branch:    status.Branch,
		PID:       status.PID,
		Question:  status.Question,
		PRURL:     status.PRURL,
		StartedAt: status.StartedAt,
		Runs:      status.Runs,
	}
}

func agenticWorkspaceStatusFromRunner(status *WorkspaceStatus) *agentic.WorkspaceStatus {
	if status == nil {
		return nil
	}
	return &agentic.WorkspaceStatus{
		Status:    status.Status,
		Agent:     status.Agent,
		Repo:      status.Repo,
		Org:       status.Org,
		Task:      status.Task,
		Branch:    status.Branch,
		PID:       status.PID,
		Question:  status.Question,
		PRURL:     status.PRURL,
		StartedAt: status.StartedAt,
		Runs:      status.Runs,
	}
}

// WorkspaceRoot returns the root directory for agent workspaces.
//
//	root := runner.WorkspaceRoot() // ~/Code/.core/workspace
func WorkspaceRoot() string {
	return agentic.WorkspaceRoot()
}

// CoreRoot returns the root directory for core ecosystem files.
//
//	root := runner.CoreRoot() // ~/Code/.core
func CoreRoot() string {
	return agentic.CoreRoot()
}

// ReadStatus reads `status.json` from one workspace directory.
//
//	st, err := runner.ReadStatus("/srv/core/workspace/core/go-io/task-5")
func ReadStatus(wsDir string) (*WorkspaceStatus, error) {
	r := ReadStatusResult(wsDir)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return nil, core.E("runner.ReadStatus", "failed to read status", nil)
		}
		return nil, err
	}
	st, ok := r.Value.(*WorkspaceStatus)
	if !ok || st == nil {
		return nil, core.E("runner.ReadStatus", "invalid status payload", nil)
	}
	return st, nil
}

// ReadStatusResult reads status.json as core.Result.
//
//	result := ReadStatusResult("/srv/core/workspace/core/go-io/task-5")
//	if result.OK { st := result.Value.(*WorkspaceStatus) }
func ReadStatusResult(wsDir string) core.Result {
	status, err := agentic.ReadStatus(wsDir)
	if err != nil {
		return core.Result{Value: core.E("runner.ReadStatusResult", "failed to read status", err), OK: false}
	}

	st := runnerWorkspaceStatusFromAgentic(status)
	if st == nil {
		return core.Result{Value: core.E("runner.ReadStatusResult", "invalid status payload", nil), OK: false}
	}
	return core.Result{Value: st, OK: true}
}

// WriteStatus writes `status.json` for one workspace directory.
//
//	r := runner.WriteStatus("/srv/core/workspace/core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
//	core.Println(r.OK)
func WriteStatus(wsDir string, status *WorkspaceStatus) core.Result {
	if status == nil {
		return core.Result{Value: core.E("runner.WriteStatus", "status is required", nil), OK: false}
	}

	agenticStatus := agenticWorkspaceStatusFromRunner(status)
	if agenticStatus == nil {
		return core.Result{Value: core.E("runner.WriteStatus", "status conversion failed", nil), OK: false}
	}
	agenticStatus.UpdatedAt = time.Now()
	if r := fs.WriteAtomic(agentic.WorkspaceStatusPath(wsDir), core.JSONMarshalString(agenticStatus)); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("runner.WriteStatus", "failed to write status", nil), OK: false}
		}
		return core.Result{Value: core.E("runner.WriteStatus", "failed to write status", err), OK: false}
	}
	return core.Result{OK: true}
}
