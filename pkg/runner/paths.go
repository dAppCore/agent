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

// ReadStatusResult reads status.json as core.Result.
//
//	result := ReadStatusResult("/srv/core/workspace/core/go-io/task-5")
//	if result.OK { workspaceStatus := result.Value.(*WorkspaceStatus) }
func ReadStatusResult(workspaceDir string) core.Result {
	statusResult := agentic.ReadStatusResult(workspaceDir)
	if !statusResult.OK {
		err, _ := statusResult.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("runner.ReadStatusResult", "failed to read status", nil), OK: false}
		}
		return core.Result{Value: core.E("runner.ReadStatusResult", "failed to read status", err), OK: false}
	}

	agenticStatus, ok := statusResult.Value.(*agentic.WorkspaceStatus)
	if !ok || agenticStatus == nil {
		return core.Result{Value: core.E("runner.ReadStatusResult", "invalid status payload", nil), OK: false}
	}
	workspaceStatus := runnerWorkspaceStatusFromAgentic(agenticStatus)
	if workspaceStatus == nil {
		return core.Result{Value: core.E("runner.ReadStatusResult", "invalid status payload", nil), OK: false}
	}
	return core.Result{Value: workspaceStatus, OK: true}
}

// WriteStatus writes `status.json` for one workspace directory.
//
//	result := runner.WriteStatus("/srv/core/workspace/core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
//	core.Println(result.OK)
func WriteStatus(workspaceDir string, status *WorkspaceStatus) core.Result {
	if status == nil {
		return core.Result{Value: core.E("runner.WriteStatus", "status is required", nil), OK: false}
	}

	agenticStatus := agenticWorkspaceStatusFromRunner(status)
	if agenticStatus == nil {
		return core.Result{Value: core.E("runner.WriteStatus", "status conversion failed", nil), OK: false}
	}
	agenticStatus.UpdatedAt = time.Now()
	if writeResult := fs.WriteAtomic(agentic.WorkspaceStatusPath(workspaceDir), core.JSONMarshalString(agenticStatus)); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("runner.WriteStatus", "failed to write status", nil), OK: false}
		}
		return core.Result{Value: core.E("runner.WriteStatus", "failed to write status", err), OK: false}
	}
	return core.Result{OK: true}
}
