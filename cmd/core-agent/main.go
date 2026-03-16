package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"forge.lthn.ai/core/mcp/pkg/mcp"
	"forge.lthn.ai/core/mcp/pkg/mcp/agentic"
	"forge.lthn.ai/core/mcp/pkg/mcp/brain"
)

func main() {
	svc, err := mcp.New(
		mcp.WithSubsystem(brain.NewDirect()),
		mcp.WithSubsystem(agentic.NewPrep()),
	)
	if err != nil {
		log.Fatalf("failed to create MCP service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := svc.Run(ctx); err != nil {
		log.Printf("MCP error: %v", err)
	}
}
