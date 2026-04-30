// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (m *Subsystem) Shutdown(ctx context.Context) error {
	return Shutdown(m, ctx)
}

func (m *Subsystem) agentStatusResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return agentStatusResource(m, ctx, request)
}
