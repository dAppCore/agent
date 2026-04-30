// SPDX-License-Identifier: EUPL-1.2

package setup

import core "dappco.re/go"

func Example_defaultBuildCommand() {
	core.Println(defaultBuildCommand(TypeGo))
	core.Println(defaultBuildCommand(TypePHP))
	// Output:
	// go build ./...
	// composer test
}

func Example_formatFlow() {
	core.Println(formatFlow(TypeNode))
	// Output:
	// - Build: `npm run build`
	// - Test: `npm test`
}

func ExampleService_Run() {
	result := (&Service{}).Run(Options{Path: "/tmp/core-agent-setup-example", DryRun: true})
	core.Println(result.OK)
	// Output:
	// Project: core-agent-setup-example
	// Type:    unknown
	//
	// Would create /tmp/core-agent-setup-example/.core/
	//   /tmp/core-agent-setup-example/.core/build.yaml
	//   /tmp/core-agent-setup-example/.core/test.yaml
	// true
}
