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

// ReadStatus reads a workspace status.json.
//
//	st, err := runner.ReadStatus("/path/to/workspace")
func ReadStatus(wsDir string) (*WorkspaceStatus, error) {
	status, err := agentic.ReadStatus(wsDir)
	if err != nil {
		return nil, core.E("runner.ReadStatus", "failed to read status", err)
	}

	json := core.JSONMarshalString(status)
	var st WorkspaceStatus
	if result := core.JSONUnmarshalString(json, &st); !result.OK {
		parseErr, _ := result.Value.(error)
		return nil, core.E("runner.ReadStatus", "failed to parse status", parseErr)
	}
	return &st, nil
}

// WriteStatus writes a workspace status.json.
//
//	runner.WriteStatus(wsDir, &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
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
