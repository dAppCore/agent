// SPDX-License-Identifier: EUPL-1.2

// Package brain provides an MCP subsystem that proxies OpenBrain knowledge
// store operations to the Laravel php-agentic backend via the IDE bridge.
package brain

import (
	"context"
	"unsafe"

	core "dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp/ide"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fs provides unrestricted filesystem access (root "/" = no sandbox).
//
//	r := fs.Read(filepath.Join(home, ".claude", "brain.key"))
//	if r.OK { apiKey = strings.TrimSpace(r.Value.(string)) }
var fs = newFs("/")

// newFs creates a core.Fs with the given root directory.
func newFs(root string) *core.Fs {
	type fsRoot struct{ root string }
	f := &core.Fs{}
	(*fsRoot)(unsafe.Pointer(f)).root = root
	return f
}

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = core.E("brain", "bridge not available", nil)

// Subsystem implements mcp.Subsystem for OpenBrain knowledge store operations.
// It proxies brain_* tool calls to the Laravel backend via the shared IDE bridge.
type Subsystem struct {
	bridge *ide.Bridge
}

// New creates a brain subsystem that uses the given IDE bridge for Laravel communication.
// Pass nil if headless (tools will return errBridgeNotAvailable).
func New(bridge *ide.Bridge) *Subsystem {
	return &Subsystem{bridge: bridge}
}

// Name implements mcp.Subsystem.
func (s *Subsystem) Name() string { return "brain" }

// RegisterTools implements mcp.Subsystem.
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerBrainTools(server)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *Subsystem) Shutdown(_ context.Context) error {
	return nil
}
