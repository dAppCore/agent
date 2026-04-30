// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func failureResult(action, fallback string, result core.Result) core.Result {
	if err, ok := result.Value.(error); ok && err != nil {
		return core.Fail(err)
	}

	message := core.Trim(stringValue(result.Value))
	if message == "" {
		message = fallback
	}

	return core.Fail(core.E(action, message, nil))
}

func typedResultValue[T any](action, invalid string, result core.Result) core.Result {
	if !result.OK {
		return result
	}

	value, ok := result.Value.(T)
	if !ok {
		return core.Fail(core.E(action, invalid, nil))
	}

	return core.Ok(value)
}

func toolHandlerFor[In, Out any](action, invalid string, fn func(context.Context, In) core.Result) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		var zero Out

		result := fn(ctx, input)
		if !result.OK {
			failed := failureResult(action, "request failed", result)
			err, _ := failed.Value.(error)
			return nil, zero, err
		}

		typed := typedResultValue[Out](action, invalid, result)
		if !typed.OK {
			err, _ := typed.Value.(error)
			return nil, zero, err
		}

		return nil, typed.Value.(Out), nil
	}
}
