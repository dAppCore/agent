// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func ExampleWorkspaceRoot() {
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}

func ExampleCoreRoot() {
	root := CoreRoot()
	core.Println(core.HasSuffix(root, ".core"))
	// Output: true
}

func ExampleHomeDir() {
	home := HomeDir()
	core.Println(home != "")
	// Output: true
}

func ExamplePlansRoot() {
	root := PlansRoot()
	core.Println(core.HasSuffix(root, "plans"))
	// Output: true
}

func ExampleAgentName() {
	name := AgentName()
	core.Println(name != "")
	// Output: true
}

func ExampleGitHubOrg() {
	org := GitHubOrg()
	core.Println(org)
	// Output: dAppCore
}

func ExampleLocalFs() {
	f := LocalFs()
	core.Println(f.Root())
	// Output: /
}
