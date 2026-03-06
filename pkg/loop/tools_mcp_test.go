package loop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMCPTools_Good_ConvertsRecords(t *testing.T) {
	handler := func(ctx context.Context, args map[string]any) (string, error) {
		return "result", nil
	}

	tool := Tool{
		Name:        "file_read",
		Description: "Read a file",
		Parameters:  map[string]any{"type": "object"},
		Handler:     handler,
	}

	assert.Equal(t, "file_read", tool.Name)
	result, err := tool.Handler(context.Background(), map[string]any{"path": "/tmp/test"})
	require.NoError(t, err)
	assert.Equal(t, "result", result)
}

func TestWrapRESTHandler_Good(t *testing.T) {
	restHandler := func(ctx context.Context, body []byte) (any, error) {
		var input map[string]any
		json.Unmarshal(body, &input)
		return map[string]string{"content": "hello from " + input["path"].(string)}, nil
	}

	wrapped := WrapRESTHandler(restHandler)
	result, err := wrapped(context.Background(), map[string]any{"path": "/tmp/test"})
	require.NoError(t, err)
	assert.Contains(t, result, "hello from /tmp/test")
}

func TestWrapRESTHandler_Bad_HandlerError(t *testing.T) {
	restHandler := func(ctx context.Context, body []byte) (any, error) {
		return nil, assert.AnError
	}

	wrapped := WrapRESTHandler(restHandler)
	_, err := wrapped(context.Background(), map[string]any{})
	require.Error(t, err)
}
