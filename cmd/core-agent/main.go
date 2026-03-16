package main

import (
	"log"

	"forge.lthn.ai/core/agent/pkg/agentic"
	"forge.lthn.ai/core/agent/pkg/brain"
	"forge.lthn.ai/core/cli/pkg/cli"
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
		svc, err := mcp.New(
			mcp.WithSubsystem(brain.NewDirect()),
			mcp.WithSubsystem(agentic.NewPrep()),
		)
		if err != nil {
			return cli.Wrap(err, "create MCP service")
		}

		return svc.Run(cmd.Context())
	})

	cli.RootCmd().AddCommand(mcpCmd)

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
