// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
)

// fs reuses the shared unrestricted filesystem used by agentic.
var fs = agentic.LocalFs()

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
		return core.Result{Value: core.E("runner.ReadStatus", "failed to read status", err), OK: false}
	}

	json := core.JSONMarshalString(status)
	var st WorkspaceStatus
	if result := core.JSONUnmarshalString(json, &st); !result.OK {
		parseErr, _ := result.Value.(error)
		if parseErr == nil {
			parseErr = core.E("runner.ReadStatus", "failed to parse status", nil)
		}
		return core.Result{Value: parseErr, OK: false}
	}
	return core.Result{Value: &st, OK: true}
}

// WriteStatus writes `status.json` for one workspace directory.
//
//	runner.WriteStatus("/srv/core/workspace/core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
func WriteStatus(wsDir string, st *WorkspaceStatus) {
	if st == nil {
		return
	}

	json := core.JSONMarshalString(st)
	var status agentic.WorkspaceStatus
	if result := core.JSONUnmarshalString(json, &status); !result.OK {
		return
	}
	status.UpdatedAt = time.Now()
	fs.WriteAtomic(agentic.WorkspaceStatusPath(wsDir), core.JSONMarshalString(&status))
}
