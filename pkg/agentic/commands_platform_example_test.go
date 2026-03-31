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

func ExamplePrepSubsystem_cmdAuthProvision() {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
	}

	result := s.cmdAuthProvision(core.NewOptions())
	core.Println(result.OK)
	// Output:
	// usage: core-agent auth provision <oauth-user-id> [--name=codex] [--permissions=plans:read,plans:write] [--rate-limit=60] [--expires-at=2026-04-01T00:00:00Z]
	// false
}
