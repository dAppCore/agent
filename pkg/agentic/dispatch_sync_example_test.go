// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Example_containerCommand() {
	cmd, args := containerCommand("codex", []string{"--model", "gpt-5.4"}, "/workspace/task-5", "/workspace/task-5/.meta")
	core.Println(cmd)
	core.Println(len(args) > 0)
	// Output:
	// docker
	// true
}

func ExamplePrepSubsystem_DispatchSync() {
	subsystem := &PrepSubsystem{}
	subsystem.dispatchSyncPrep = func(context.Context, *mcp.CallToolRequest, PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{}, core.E("dispatchSync", "boom", nil)
	}
	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{Repo: "go-io", Task: "Fix tests"})
	core.Println(result.OK)
	// Output: false
}
