// SPDX-License-Identifier: EUPL-1.2

// Package brain gives MCP and HTTP services a shared OpenBrain surface.
//
//	sub := brain.New(bridge)
//	sub.RegisterTools(server)
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
//	keyPath := core.Concat(home, "/.claude/brain.key")
//	if r := fs.Read(keyPath); r.OK {
//		apiKey = core.Trim(r.Value.(string))
//	}
var fs = agentic.LocalFs()

func fieldString(values map[string]any, key string) string {
	return core.Sprint(values[key])
}

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = core.E("brain", "bridge not available", nil)

// Subsystem routes brain_* MCP tools through the shared IDE bridge.
//
//	sub := brain.New(bridge)
//	sub.RegisterTools(server)
type Subsystem struct {
	bridge *ide.Bridge
}

// New builds the bridge-backed OpenBrain subsystem.
//
//	sub := brain.New(bridge)
//	_ = sub.Shutdown(context.Background())
func New(bridge *ide.Bridge) *Subsystem {
	return &Subsystem{bridge: bridge}
}

// Name keeps the subsystem address stable for core.WithService and MCP.
//
//	name := sub.Name() // "brain"
func (s *Subsystem) Name() string { return "brain" }

// RegisterTools publishes the bridge-backed brain tools on an MCP server.
//
//	sub := brain.New(bridge)
//	sub.RegisterTools(server)
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerBrainTools(server)
}

// Shutdown satisfies the MCP subsystem lifecycle without extra cleanup.
//
//	_ = sub.Shutdown(context.Background())
func (s *Subsystem) Shutdown(_ context.Context) error {
	return nil
}
