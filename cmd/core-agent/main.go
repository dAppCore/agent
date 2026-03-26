package main

import (
	"dappco.re/go/core"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/mcp/pkg/mcp"
)

func main() {
	// Set log level early — before ServiceStartup to suppress startup noise.
	// --quiet/-q reduces to errors only, --debug shows everything.
	applyLogLevel()

	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
		core.WithService(mcp.Register),
	)
	c.App().Version = appVersion()

	c.Cli().SetBanner(func(_ *core.Cli) string {
		return core.Concat("core-agent ", c.App().Version, " — agentic orchestration for the Core ecosystem")
	})

	registerAppCommands(c)

	c.Run()
}

// appVersion returns the build version or "dev".
func appVersion() string {
	if version != "" {
		return version
	}
	return "dev"
}
