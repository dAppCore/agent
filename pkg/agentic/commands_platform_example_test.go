// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func ExamplePrepSubsystem_cmdFleetRegister() {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
	}

	result := s.cmdFleetRegister(core.NewOptions())
	core.Println(result.OK)
	// Output:
	// usage: core-agent fleet register <agent-id> --platform=linux [--models=codex,gpt-5.4] [--capabilities=go,review]
	// false
}
