// SPDX-License-Identifier: EUPL-1.2

// subsystem := brain.New(nil)
// core.Println(subsystem.Name()) // "brain"
package brain

import (
	"context"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp/ide"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// keyPath := core.JoinPath(home, ".claude", "brain.key")
//
//	if readResult := fs.Read(keyPath); readResult.OK {
//		apiKey = core.Trim(readResult.Value.(string))
//	}
var fs = agentic.LocalFs()

func stringField(values map[string]any, key string) string {
	return core.Sprint(values[key])
}

// core.E("brain", "bridge not available", nil)
var errBridgeNotAvailable = core.E("brain", "bridge not available", nil)

// subsystem := brain.New(nil)
// core.Println(subsystem.Name()) // "brain"
type Subsystem struct {
	bridge *ide.Bridge
}

// subsystem := brain.New(nil)
// core.Println(subsystem.Name())
func New(bridge *ide.Bridge) *Subsystem {
	return &Subsystem{bridge: bridge}
}

// name := subsystem.Name() // "brain"
func (s *Subsystem) Name() string { return "brain" }

// subsystem := brain.New(nil)
// subsystem.RegisterTools(server)
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerBrainTools(server)
}

// _ = subsystem.Shutdown(context.Background())
func (s *Subsystem) Shutdown(_ context.Context) error {
	return nil
}
