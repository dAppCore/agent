package loop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEaaSTools_Good_ReturnsThreeTools(t *testing.T) {
	tools := EaaSTools("http://localhost:8009")
	assert.Len(t, tools, 3)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	assert.Contains(t, names, "eaas_score")
	assert.Contains(t, names, "eaas_imprint")
	assert.Contains(t, names, "eaas_similar")
}

func TestEaaSScore_Good_CallsAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/score/content", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"verdict": "likely_human",
			"lek":     85.5,
		})
	}))
	defer server.Close()

	tools := EaaSTools(server.URL)
	var scoreTool Tool
	for _, tool := range tools {
		if tool.Name == "eaas_score" {
			scoreTool = tool
			break
		}
	}

	result, err := scoreTool.Handler(context.Background(), map[string]any{"text": "Hello world"})
	require.NoError(t, err)
	assert.Contains(t, result, "likely_human")
}
