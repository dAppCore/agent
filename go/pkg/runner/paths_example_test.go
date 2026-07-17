// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	core "dappco.re/go"
)

func ExampleCoreRoot() {
	root := CoreRoot()
	core.Println(core.HasSuffix(root, "data"))
	// Output: true
}

func ExampleWorkspaceRoot() {
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}

func ExampleReadStatusResult() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := core.MustCast[string](fsys.TempDir("runner-paths-result"))
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
	dir := core.MustCast[string](fsys.TempDir("runner-paths"))
	defer fsys.DeleteAll(dir)

	result := WriteStatus(dir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
	})
	core.Println(result.OK)

	statusResult := ReadStatusResult(dir)
	core.Println(statusResult.OK)
	core.Println(statusResult.Value.(*WorkspaceStatus).Status)
	// Output:
	// true
	// true
	// running
}
