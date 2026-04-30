// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func monitorExampleResourceCount(register func(*coremcp.Service)) int {
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

	result, err := clientSession.ListResources(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	return len(result.Resources)
}

func ExampleNew() {
	mon := New(Options{Interval: 30 * time.Second})
	core.Println(mon.Name())
	// Output: monitor
}

func ExampleRegister() {
	c := core.New(core.WithService(Register))

	service := c.Service("monitor")
	svc, ok := service.Value.(*Subsystem)
	core.Println(ok)
	core.Println(svc.Name())
	// Output:
	// true
	// monitor
}

func ExampleSubsystem_HandleIPCEvents() {
	result := New().HandleIPCEvents(nil, "unknown")
	core.Println(result.OK)
	// Output: true
}

func ExampleSubsystem_Name() {
	core.Println(New().Name())
	// Output: monitor
}

func ExampleSubsystem_RegisterTools() {
	core.Println(monitorExampleResourceCount(New().RegisterTools))
	// Output: 1
}

func ExampleSubsystem_Start() {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	core.Println(mon.done != nil)
	_ = mon.Shutdown(context.Background())
	// Output: true
}

func ExampleSubsystem_OnStartup() {
	mon := New(Options{Interval: time.Hour})
	core.Println(mon.OnStartup(context.Background()).OK)
	_ = mon.Shutdown(context.Background())
	// Output: true
}

func ExampleSubsystem_OnShutdown() {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	core.Println(mon.OnShutdown(context.Background()).OK)
	// Output: true
}

func ExampleSubsystem_Shutdown() {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	core.Println(mon.Shutdown(context.Background()) == nil)
	// Output: true
}

func ExampleSubsystem_Poke() {
	mon := New()
	mon.Poke()
	core.Println(len(mon.poke))
	// Output: 1
}
