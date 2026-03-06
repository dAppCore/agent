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

func TestClient_Remember_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/brain/remember", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req RememberRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Go uses structural typing", req.Content)
		assert.Equal(t, "fact", req.Type)
		assert.Equal(t, "go-agentic", req.Project)
		assert.Equal(t, []string{"go", "typing"}, req.Tags)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(RememberResponse{
			ID:        "mem-abc-123",
			Type:      "fact",
			Project:   "go-agentic",
			CreatedAt: "2026-03-03T12:00:00+00:00",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.Remember(context.Background(), RememberRequest{
		Content: "Go uses structural typing",
		Type:    "fact",
		Project: "go-agentic",
		Tags:    []string{"go", "typing"},
	})

	require.NoError(t, err)
	assert.Equal(t, "mem-abc-123", result.ID)
	assert.Equal(t, "fact", result.Type)
	assert.Equal(t, "go-agentic", result.Project)
}

func TestClient_Remember_Bad_EmptyContent(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	result, err := client.Remember(context.Background(), RememberRequest{
		Type: "fact",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "content is required")
}

func TestClient_Remember_Bad_EmptyType(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	result, err := client.Remember(context.Background(), RememberRequest{
		Content: "something",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "type is required")
}

func TestClient_Remember_Bad_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(APIError{Message: "validation failed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.Remember(context.Background(), RememberRequest{
		Content: "test",
		Type:    "fact",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestClient_Recall_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/brain/recall", r.URL.Path)

		var req RecallRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "how does typing work in Go", req.Query)
		assert.Equal(t, 5, req.TopK)
		assert.Equal(t, "go-agentic", req.Project)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RecallResponse{
			Memories: []Memory{
				{
					ID:         "mem-abc-123",
					Type:       "fact",
					Content:    "Go uses structural typing",
					Project:    "go-agentic",
					Confidence: 0.95,
				},
			},
			Scores: map[string]float64{
				"mem-abc-123": 0.87,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.Recall(context.Background(), RecallRequest{
		Query:   "how does typing work in Go",
		TopK:    5,
		Project: "go-agentic",
	})

	require.NoError(t, err)
	assert.Len(t, result.Memories, 1)
	assert.Equal(t, "mem-abc-123", result.Memories[0].ID)
	assert.Equal(t, "Go uses structural typing", result.Memories[0].Content)
	assert.InDelta(t, 0.87, result.Scores["mem-abc-123"], 0.001)
}

func TestClient_Recall_Good_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RecallResponse{
			Memories: []Memory{},
			Scores:   map[string]float64{},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.Recall(context.Background(), RecallRequest{
		Query: "something obscure",
	})

	require.NoError(t, err)
	assert.Empty(t, result.Memories)
	assert.Empty(t, result.Scores)
}

func TestClient_Recall_Bad_EmptyQuery(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	result, err := client.Recall(context.Background(), RecallRequest{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "query is required")
}

func TestClient_Forget_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v1/brain/forget/mem-abc-123", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.Forget(context.Background(), "mem-abc-123")

	assert.NoError(t, err)
}

func TestClient_Forget_Bad_EmptyID(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	err := client.Forget(context.Background(), "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "memory ID is required")
}

func TestClient_Forget_Bad_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(APIError{Message: "memory not found"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.Forget(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "memory not found")
}

func TestClient_EnsureCollection_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/brain/collections", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.EnsureCollection(context.Background())

	assert.NoError(t, err)
}

func TestClient_EnsureCollection_Bad_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIError{Message: "collection setup failed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.EnsureCollection(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collection setup failed")
}
