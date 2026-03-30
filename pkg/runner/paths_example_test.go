// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	core "dappco.re/go/core"
)

func ExampleCoreRoot() {
	root := CoreRoot()
	core.Println(core.HasSuffix(root, ".core"))
	// Output: true
}

func ExampleWorkspaceRoot() {
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}

func ExampleReadStatus() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := fsys.TempDir("runner-paths-read")
	defer fsys.DeleteAll(dir)

	WriteStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
	})

	st, err := ReadStatus(dir)
	core.Println(err == nil)
	core.Println(st.Repo)
	// Output:
	// true
	// go-io
}

func ExampleReadStatusResult() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := fsys.TempDir("runner-paths-result")
	defer fsys.DeleteAll(dir)

	WriteStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
	})

	result := ReadStatusResult(dir)
	core.Println(result.OK)
	core.Println(result.Value.(*WorkspaceStatus).Repo)
	// Output:
	// true
	// go-io
}

func ExampleWriteStatus() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := fsys.TempDir("runner-paths")
	defer fsys.DeleteAll(dir)

	WriteStatus(dir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
	})

	st, err := ReadStatus(dir)
	core.Println(err == nil)
	core.Println(st.Status)
	// Output:
	// true
	// running
}
