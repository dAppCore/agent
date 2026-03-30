// SPDX-License-Identifier: EUPL-1.2

// Package brain gives MCP and HTTP services the same OpenBrain capability map.
//
//	subsystem := brain.New(nil)
//	core.Println(subsystem.Name())
package brain

import (
	"context"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp/ide"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fs provides unrestricted filesystem access for shared brain credentials.
//
//	keyPath := core.JoinPath(home, ".claude", "brain.key")
//	if readResult := fs.Read(keyPath); readResult.OK {
//		apiKey = core.Trim(readResult.Value.(string))
//	}
var fs = agentic.LocalFs()

func stringField(values map[string]any, key string) string {
	return core.Sprint(values[key])
}

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = core.E("brain", "bridge not available", nil)

// Subsystem routes `brain_*` MCP tools through the shared IDE bridge.
//
//	subsystem := brain.New(nil)
//	core.Println(subsystem.Name()) // "brain"
type Subsystem struct {
	bridge *ide.Bridge
}

// New builds the bridge-backed OpenBrain subsystem used by MCP.
//
//	subsystem := brain.New(nil)
//	core.Println(subsystem.Name())
func New(bridge *ide.Bridge) *Subsystem {
	return &Subsystem{bridge: bridge}
}

// Name keeps the subsystem address stable for core.WithService and MCP.
//
//	name := subsystem.Name() // "brain"
func (s *Subsystem) Name() string { return "brain" }

// RegisterTools publishes the bridge-backed brain tools on an MCP server.
//
//	subsystem := brain.New(nil)
//	subsystem.RegisterTools(server)
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerBrainTools(server)
}

// Shutdown satisfies the MCP subsystem lifecycle without extra cleanup.
//
//	_ = subsystem.Shutdown(context.Background())
func (s *Subsystem) Shutdown(_ context.Context) error {
	return nil
}
