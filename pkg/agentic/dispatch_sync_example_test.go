// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_containerCommand() {
	cmd, args := containerCommand("codex", []string{"--model", "gpt-5.4"}, "/workspace/task-5", "/workspace/task-5/.meta")
	core.Println(cmd)
	core.Println(len(args) > 0)
	// Output:
	// docker
	// true
}
