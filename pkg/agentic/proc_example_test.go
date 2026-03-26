// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

func ExamplePrepSubsystem_runCmd() {
	r := testPrep.runCmd(context.Background(), ".", "echo", "hello")
	core.Println(r.OK)
	// Output: true
}

func ExamplePrepSubsystem_gitCmd() {
	r := testPrep.gitCmd(context.Background(), ".", "--version")
	core.Println(r.OK)
	// Output: true
}

func ExamplePrepSubsystem_gitOutput() {
	version := testPrep.gitOutput(context.Background(), ".", "--version")
	core.Println(core.HasPrefix(version, "git version"))
	// Output: true
}

func ExamplePrepSubsystem_runCmdOK() {
	ok := testPrep.runCmdOK(context.Background(), ".", "echo", "test")
	core.Println(ok)
	// Output: true
}

func ExamplePrepSubsystem_gitCmdOK() {
	ok := testPrep.gitCmdOK(context.Background(), ".", "--version")
	core.Println(ok)
	// Output: true
}
