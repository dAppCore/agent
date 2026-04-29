// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func brainExampleToolCount(register func(*coremcp.Service)) int {
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

func ExampleNew() {
	sub := New(nil)
	core.Println(sub.Name())
	// Output: brain
}

func ExampleRegister_services() {
	c := core.New(core.WithService(Register))
	core.Println(c.Services())
}

func ExampleSubsystem_Name() {
	core.Println(New(nil).Name())
	// Output: brain
}

func ExampleSubsystem_RegisterTools() {
	core.Println(brainExampleToolCount(New(nil).RegisterTools) > 0)
	// Output: true
}

func ExampleSubsystem_Shutdown() {
	core.Println(New(nil).Shutdown(context.Background()) == nil)
	// Output: true
}
