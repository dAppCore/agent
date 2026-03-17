package main

import (
	"log"

	"forge.lthn.ai/core/agent/pkg/agentic"
	"forge.lthn.ai/core/agent/pkg/brain"
	"forge.lthn.ai/core/agent/pkg/monitor"
	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-process"
	"forge.lthn.ai/core/go/pkg/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
)

func main() {
	if err := cli.Init(cli.Options{
		AppName: "core-agent",
		Version: "0.1.0",
	}); err != nil {
		log.Fatal(err)
	}

	mcpCmd := cli.NewCommand("mcp", "Start the MCP server on stdio", "", func(cmd *cli.Command, args []string) error {
		// Initialise go-process so dispatch can spawn agents
		c, err := core.New(core.WithName("process", process.NewService(process.Options{})))
		if err != nil {
			return cli.Wrap(err, "init core")
		}
		procSvc, err := core.ServiceFor[*process.Service](c, "process")
		if err != nil {
			return cli.Wrap(err, "get process service")
		}
		process.SetDefault(procSvc)

		mon := monitor.New()
		mcpSvc, err := mcp.New(
			mcp.WithSubsystem(brain.NewDirect()),
			mcp.WithSubsystem(agentic.NewPrep()),
			mcp.WithSubsystem(mon),
		)
		if err != nil {
			return cli.Wrap(err, "create MCP service")
		}

		// Start background monitor after MCP server is running
		mon.Start(cmd.Context())

		return mcpSvc.Run(cmd.Context())
	})

	cli.RootCmd().AddCommand(mcpCmd)

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
