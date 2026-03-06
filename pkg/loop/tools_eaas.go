package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var eaasClient = &http.Client{Timeout: 30 * time.Second}

// EaaSTools returns the three EaaS tool wrappers: score, imprint, and atlas similar.
func EaaSTools(baseURL string) []Tool {
	return []Tool{
		{
			Name:        "eaas_score",
			Description: "Score text for AI-generated content, sycophancy, and compliance markers. Returns verdict, LEK score, heuristic breakdown, and detected flags.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string", "description": "Text to analyse"},
				},
				"required": []any{"text"},
			},
			Handler: eaasPostHandler(baseURL, "/v1/score/content"),
		},
		{
			Name:        "eaas_imprint",
			Description: "Analyse the linguistic imprint of text. Returns stylistic fingerprint metrics.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string", "description": "Text to analyse"},
				},
				"required": []any{"text"},
			},
			Handler: eaasPostHandler(baseURL, "/v1/score/imprint"),
		},
		{
			Name:        "eaas_similar",
			Description: "Find similar previously scored content via atlas vector search.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string", "description": "Scoring ID to search from"},
					"limit": map[string]any{"type": "integer", "description": "Max results (default 5)"},
				},
				"required": []any{"id"},
			},
			Handler: eaasPostHandler(baseURL, "/v1/atlas/similar"),
		},
	}
}

func eaasPostHandler(baseURL, path string) func(context.Context, map[string]any) (string, error) {
	return func(ctx context.Context, args map[string]any) (string, error) {
		body, err := json.Marshal(args)
		if err != nil {
			return "", fmt.Errorf("marshal args: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", baseURL+path, bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := eaasClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("eaas request: %w", err)
		}
		defer resp.Body.Close()

		result, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("eaas returned %d: %s", resp.StatusCode, string(result))
		}

		return string(result), nil
	}
}
