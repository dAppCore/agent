package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	srv := newServer()
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("mcp server failed: %v", err)
	}
}
