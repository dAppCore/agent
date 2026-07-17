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

func ExampleMountData() {
	c := core.New()
	MountData(c)
	r := c.Data().ReadString("prompts/coding.md")
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

func ExampleTemplate() {
	r := Template("coding")
	core.Println(r.OK)
	// Output: true
}

func ExampleTaskBundle() {
	r := TaskBundle("code/review")
	bundle := r.Value.(Bundle)
	core.Println(r.OK)
	core.Println(bundle.Main != "")
	core.Println(len(bundle.Files) > 0)
	// Output:
	// true
	// true
	// true
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

func ExampleListPersonas() {
	personas := ListPersonas()
	core.Println(len(personas) > 0)
	// Output: true
}

func ExampleListWorkspaces() {
	workspaces := ListWorkspaces()
	core.Println(len(workspaces) > 0)
	// Output: true
}

func ExamplePersona() {
	personas := ListPersonas()
	r := Persona(personas[0])
	core.Println(r.OK)
	// Output: true
}

func ExampleExtractWorkspace() {
	fsys := (&core.Fs{}).NewUnrestricted()
	dir := "/tmp/core-agent-lib-example"
	defer fsys.DeleteAll(dir)

	result := ExtractWorkspace("default", dir, &WorkspaceData{
		Repo:  "go-io",
		Task:  "Fix tests",
		Agent: "codex",
	})
	core.Println(result.OK)
	core.Println(fsys.Exists(core.JoinPath(dir, "CODEX.md")).OK)
	// Output:
	// true
	// true
}

func ExampleWorkspaceFile() {
	r := WorkspaceFile("default", "CODEX.md.tmpl")
	core.Println(r.OK)
	// Output: true
}
