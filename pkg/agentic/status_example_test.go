// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"time"

	core "dappco.re/go/core"
)

func Example_writeStatus() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example-ws")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     "codex",
		Repo:      "go-io",
		Task:      "Fix tests",
		StartedAt: time.Now(),
	}
	err := writeStatus(dir, st)
	core.Println(err == nil)
	// Output: true
}

func ExampleReadStatusResult() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := fsys.TempDir("agentic-status-result")
	defer fsys.DeleteAll(dir)

	status := &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
	}
	core.Println(fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	result := ReadStatusResult(dir)
	core.Println(result.OK)
	core.Println(result.Value.(*WorkspaceStatus).Repo)
	// Output:
	// true
	// true
	// go-io
}

func ExampleReadStatus() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := fsys.TempDir("agentic-status")
	defer fsys.DeleteAll(dir)

	status := &WorkspaceStatus{
		Status: "blocked",
		Agent:  "claude",
		Repo:   "go-io",
	}
	core.Println(fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	read, err := ReadStatus(dir)
	core.Println(err == nil)
	core.Println(read.Repo)
	// Output:
	// true
	// true
	// go-io
}
