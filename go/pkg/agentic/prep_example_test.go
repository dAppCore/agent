// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func prepExampleToolCount(register func(*coremcp.Service)) int {
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	if err != nil {
		panic(err)
	}
	register(svc)
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "example", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := svc.Server().Connect(context.Background(), serverTransport, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = serverSession.Close() }()
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = clientSession.Close() }()
	result, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	return len(result.Tools)
}

func ExampleAgentOptions() {
	opts := AgentOptions{}
	core.Println(opts == AgentOptions{})
	// Output: true
}

func ExamplePrepInput() {
	input := PrepInput{Repo: "go-io", Issue: 42}
	core.Println(input.Repo, input.Issue)
	// Output: go-io 42
}

func ExampleNewPrep() {
	prep := NewPrep()
	core.Println(prep != nil)
	// Output: true
}

func ExamplePrepSubsystem_OnStartup() {
	subsystem := NewPrep()
	subsystem.ServiceRuntime = core.NewServiceRuntime(core.New(), AgentOptions{})
	core.Println(subsystem.OnStartup(context.Background()).OK)
	_ = subsystem.OnShutdown(context.Background())
	// Output: true
}

func ExamplePrepSubsystem_OnShutdown() {
	core.Println(NewPrep().OnShutdown(context.Background()).OK)
	// Output: true
}

func ExamplePrepSubsystem_TrackWorkspace() {
	subsystem := NewPrep()
	subsystem.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "queued"})
	core.Println(subsystem.Workspaces().Len())
	// Output: 1
}

func ExamplePrepSubsystem_Workspaces() {
	core.Println(NewPrep().Workspaces().Len())
	// Output: 0
}

func ExamplePrepSubsystem_Name() {
	core.Println(NewPrep().Name())
	// Output: agentic
}

func ExamplePrepSubsystem_SetCore() {
	subsystem := &PrepSubsystem{}
	subsystem.SetCore(core.New())
	core.Println(subsystem.Core() != nil)
	// Output: true
}

func ExamplePrepSubsystem_RegisterTools() {
	core.Println(prepExampleToolCount(NewPrep().RegisterTools) > 0)
	// Output: true
}

func ExamplePrepSubsystem_Shutdown() {
	core.Println(NewPrep().Shutdown(context.Background()) == nil)
	// Output: true
}

func ExamplePrepSubsystem_PrepareWorkspace() {
	_, _, err := NewPrep().PrepareWorkspace(context.Background(), PrepInput{})
	core.Println(err != nil)
	// Output: true
}

func ExamplePrepSubsystem_TestPrepWorkspace() {
	_, _, err := NewPrep().TestPrepWorkspace(context.Background(), PrepInput{})
	core.Println(err != nil)
	// Output: true
}

func ExamplePrepSubsystem_BuildPrompt() {
	subsystem := NewPrep()
	subsystem.ServiceRuntime = core.NewServiceRuntime(core.New(), AgentOptions{})
	prompt, _, _ := subsystem.BuildPrompt(context.Background(), PrepInput{Org: "core", Repo: "go-io", Task: "Fix tests"}, "dev", ".")
	core.Println(core.Contains(prompt, "TASK: Fix tests"))
	// Output: true
}

func ExamplePrepSubsystem_TestBuildPrompt() {
	subsystem := NewPrep()
	subsystem.ServiceRuntime = core.NewServiceRuntime(core.New(), AgentOptions{})
	prompt, _, _ := subsystem.TestBuildPrompt(context.Background(), PrepInput{Org: "core", Repo: "go-io", Task: "Fix tests"}, "dev", ".")
	core.Println(core.Contains(prompt, "TASK: Fix tests"))
	// Output: true
}
