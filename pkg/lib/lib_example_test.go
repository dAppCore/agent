// SPDX-License-Identifier: EUPL-1.2

package lib

import (
	core "dappco.re/go"
)

func ExamplePrompt() {
	r := Prompt("coding")
	core.Println(r.OK)
	// Output: true
}

func ExampleTask() {
	r := Task("bug-fix")
	core.Println(r.OK)
	// Output: true
}

func ExampleFlow() {
	r := Flow("go")
	core.Println(r.OK)
	// Output: true
}

func ExampleListPrompts() {
	prompts := ListPrompts()
	core.Println(len(prompts) > 0)
	// Output: true
}

func ExampleListFlows() {
	flows := ListFlows()
	core.Println(len(flows) > 0)
	// Output: true
}

func ExampleListTasks() {
	tasks := ListTasks()
	core.Println(len(tasks) > 0)
	// Output: true
}

func ExampleListWorkspaces() {
	workspaces := ListWorkspaces()
	core.Println(len(workspaces) > 0)
	// Output: true
}

func ExampleListPersonas() {
	personas := ListPersonas()
	core.Println(len(personas) > 0)
	// Output: true
}

func ExampleTemplate() {
	r := Template("coding")
	core.Println(r.OK)
	// Output: true
}

func ExampleWorkspaceFile() {
	r := WorkspaceFile("default", "CODEX.md.tmpl")
	core.Println(r.OK)
	// Output: true
}

func ExampleMountData() {
	c := core.New()
	MountData(c)

	// Other services can now access content via Core
	r := c.Data().ReadString("prompts/coding.md")
	core.Println(r.OK)
	// Output: true
}

func ExampleTaskBundle() {
	r := TaskBundle("code/review")
	if r.OK {
		b := r.Value.(Bundle)
		core.Println(b.Main != "")
		core.Println(len(b.Files) > 0)
	}
	// Output:
	// true
	// true
}

func ExampleWorkspaceData() {
	data := &WorkspaceData{
		Repo:     "go-io",
		Task:     "fix tests",
		Agent:    "codex",
		BuildCmd: "go build ./...",
	}
	core.Println(data.Repo, data.Agent, data.BuildCmd)
	// Output: go-io codex go build ./...
}

func ExampleExtractWorkspace() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example-ws")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	r := ExtractWorkspace("default", dir, &WorkspaceData{
		Repo: "go-io", Task: "fix tests",
	})
	core.Println(r.OK)
	// Output: true
}
