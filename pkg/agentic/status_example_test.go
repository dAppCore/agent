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

func ExampleReadStatus() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example-ws")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "claude",
		Repo:   "go-io",
	})

	st, err := ReadStatus(dir)
	core.Println(err == nil)
	core.Println(st.Status)
	core.Println(st.Agent)
	// Output:
	// true
	// completed
	// claude
}
