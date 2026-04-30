// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

func Example_resolveWorkspace() {
	resolved := resolveWorkspace("nonexistent/workspace")
	core.Println(resolved == "")
	// Output: true
}

func ExampleRegisterHandlers() {
	RegisterHandlers(core.New(), &PrepSubsystem{})
	core.Println(true)
	// Output: true
}

func ExamplePrepSubsystem_HandleIPCEvents() {
	result := (&PrepSubsystem{}).HandleIPCEvents(core.New(), messages.PokeQueue{})
	core.Println(result.OK)
	// Output: true
}

func ExamplePrepSubsystem_SpawnFromQueue() {
	result := (&PrepSubsystem{}).SpawnFromQueue("unknown", "Write docs", "/tmp/example-workspace")
	core.Println(result.OK)
	// Output: false
}
