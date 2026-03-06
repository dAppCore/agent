package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ScoreContent_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/score/content", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var req ScoreContentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Contains(t, req.Text, "sample text for scoring")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"score":      0.23,
			"confidence": 0.91,
			"label":      "human",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ScoreContent(context.Background(), ScoreContentRequest{
		Text: "This is some sample text for scoring that is at least twenty characters",
	})

	require.NoError(t, err)
	assert.InDelta(t, 0.23, result["score"], 0.001)
	assert.Equal(t, "human", result["label"])
}

func TestClient_ScoreContent_Good_WithPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ScoreContentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Check for formality", req.Prompt)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"score": 0.5})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ScoreContent(context.Background(), ScoreContentRequest{
		Text:   "This text should be checked for formality and style patterns",
		Prompt: "Check for formality",
	})

	require.NoError(t, err)
	assert.InDelta(t, 0.5, result["score"], 0.001)
}

func TestClient_ScoreContent_Bad_EmptyText(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	result, err := client.ScoreContent(context.Background(), ScoreContentRequest{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "text is required")
}

func TestClient_ScoreContent_Bad_ServiceDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   "scoring_unavailable",
			"message": "Could not reach the scoring service.",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ScoreContent(context.Background(), ScoreContentRequest{
		Text: "This text needs at least twenty characters to validate",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_ScoreImprint_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/score/imprint", r.URL.Path)

		var req ScoreImprintRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.NotEmpty(t, req.Text)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"imprint":    "abc123def456",
			"confidence": 0.88,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ScoreImprint(context.Background(), ScoreImprintRequest{
		Text: "This text has a distinct linguistic imprint pattern to analyse",
	})

	require.NoError(t, err)
	assert.Equal(t, "abc123def456", result["imprint"])
}

func TestClient_ScoreImprint_Bad_EmptyText(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	result, err := client.ScoreImprint(context.Background(), ScoreImprintRequest{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "text is required")
}

func TestClient_ScoreHealth_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/score/health", r.URL.Path)
		// Health endpoint should not require auth token
		assert.Empty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ScoreHealthResponse{
			Status: "healthy",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ScoreHealth(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "healthy", result.Status)
}

func TestClient_ScoreHealth_Bad_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(ScoreHealthResponse{
			Status:         "unhealthy",
			UpstreamStatus: 503,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ScoreHealth(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
}
