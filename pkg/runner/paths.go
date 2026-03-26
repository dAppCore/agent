// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	core "dappco.re/go/core"
)

// fs is the file I/O medium for the runner package.
var fs = (&core.Fs{}).NewUnrestricted()

// WorkspaceRoot returns the root directory for agent workspaces.
//
//	root := runner.WorkspaceRoot() // ~/Code/.core/workspace
func WorkspaceRoot() string {
	return core.JoinPath(CoreRoot(), "workspace")
}

// CoreRoot returns the root directory for core ecosystem files.
//
//	root := runner.CoreRoot() // ~/Code/.core
func CoreRoot() string {
	if root := core.Env("CORE_WORKSPACE"); root != "" {
		return root
	}
	return core.JoinPath(core.Env("DIR_HOME"), "Code", ".core")
}

// ReadStatus reads a workspace status.json.
//
//	st, err := runner.ReadStatus("/path/to/workspace")
func ReadStatus(wsDir string) (*WorkspaceStatus, error) {
	path := core.JoinPath(wsDir, "status.json")
	r := fs.Read(path)
	if !r.OK {
		return nil, core.E("runner.ReadStatus", "failed to read status", nil)
	}
	var st WorkspaceStatus
	if result := core.JSONUnmarshalString(r.Value.(string), &st); !result.OK {
		return nil, core.E("runner.ReadStatus", "failed to parse status", nil)
	}
	return &st, nil
}

// WriteStatus writes a workspace status.json.
//
//	runner.WriteStatus(wsDir, &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
func WriteStatus(wsDir string, st *WorkspaceStatus) {
	path := core.JoinPath(wsDir, "status.json")
	fs.Write(path, core.JSONMarshalString(st))
}
