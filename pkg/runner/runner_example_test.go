// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

func ExampleNew() {
	svc := New()
	core.Println(svc.Workspaces().Len())
	// Output: 0
}

func ExampleRegister() {
	c := core.New(core.WithOption("name", "runner-example"))
	r := Register(c)
	core.Println(r.OK)
	// Output: true
}

func ExampleService_OnStartup() {
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(core.New(), Options{})
	core.Println(svc.OnStartup(context.Background()).OK)
	_ = svc.OnShutdown(context.Background())
	// Output: true
}

func ExampleService_OnShutdown() {
	svc := New()
	core.Println(svc.OnShutdown(context.Background()).OK)
	// Output: true
}

func ExampleService_HandleIPCEvents() {
	svc := New()
	core.Println(svc.HandleIPCEvents(core.New(), messages.PokeQueue{}).OK)
	// Output: true
}

func ExampleService_IsFrozen() {
	core.Println(New().IsFrozen())
	// Output: false
}

func ExampleService_Poke() {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	core.Println(len(svc.pokeCh))
	// Output: 1
}

func ExampleService_TrackWorkspace() {
	svc := New()
	svc.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "running", Agent: "codex"})
	core.Println(svc.Workspaces().Len())
	// Output: 1
}

func ExampleService_Workspaces() {
	core.Println(New().Workspaces().Len())
	// Output: 0
}
