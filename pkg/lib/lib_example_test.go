// SPDX-License-Identifier: EUPL-1.2

package lib

import (
	core "dappco.re/go/core"
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
