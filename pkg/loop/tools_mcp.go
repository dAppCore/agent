package loop

import (
	"context"
	"encoding/json"
	"fmt"

	aimcp "forge.lthn.ai/core/go-ai/mcp"
)

// LoadMCPTools converts all tools from a go-ai MCP Service into loop.Tool values.
func LoadMCPTools(svc *aimcp.Service) []Tool {
	var tools []Tool
	for _, record := range svc.Tools() {
		tools = append(tools, Tool{
			Name:        record.Name,
			Description: record.Description,
			Parameters:  record.InputSchema,
			Handler:     WrapRESTHandler(RESTHandlerFunc(record.RESTHandler)),
		})
	}
	return tools
}

// RESTHandlerFunc matches go-ai's mcp.RESTHandler signature.
type RESTHandlerFunc func(ctx context.Context, body []byte) (any, error)

// WrapRESTHandler converts a go-ai RESTHandler into a loop.Tool handler.
func WrapRESTHandler(handler RESTHandlerFunc) func(context.Context, map[string]any) (string, error) {
	return func(ctx context.Context, args map[string]any) (string, error) {
		body, err := json.Marshal(args)
		if err != nil {
			return "", fmt.Errorf("marshal args: %w", err)
		}

		result, err := handler(ctx, body)
		if err != nil {
			return "", err
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("marshal result: %w", err)
		}
		return string(out), nil
	}
}
