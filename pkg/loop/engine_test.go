package loop

import (
	"context"
	"iter"
	"testing"
	"time"

	"forge.lthn.ai/core/go-inference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockModel returns canned responses. Each call to Generate pops the next response.
type mockModel struct {
	responses []string
	callCount int
	lastErr   error
}

func (m *mockModel) Generate(ctx context.Context, prompt string, opts ...inference.GenerateOption) iter.Seq[inference.Token] {
	return func(yield func(inference.Token) bool) {
		if m.callCount >= len(m.responses) {
			return
		}
		resp := m.responses[m.callCount]
		m.callCount++
		for i, ch := range resp {
			if !yield(inference.Token{ID: int32(i), Text: string(ch)}) {
				return
			}
		}
	}
}

func (m *mockModel) Chat(ctx context.Context, messages []inference.Message, opts ...inference.GenerateOption) iter.Seq[inference.Token] {
	return m.Generate(ctx, "", opts...)
}

func (m *mockModel) Err() error                        { return m.lastErr }
func (m *mockModel) Close() error                      { return nil }
func (m *mockModel) ModelType() string                 { return "mock" }
func (m *mockModel) Info() inference.ModelInfo          { return inference.ModelInfo{} }
func (m *mockModel) Metrics() inference.GenerateMetrics { return inference.GenerateMetrics{} }
func (m *mockModel) Classify(ctx context.Context, p []string, o ...inference.GenerateOption) ([]inference.ClassifyResult, error) {
	return nil, nil
}
func (m *mockModel) BatchGenerate(ctx context.Context, p []string, o ...inference.GenerateOption) ([]inference.BatchResult, error) {
	return nil, nil
}

func TestEngine_Good_SimpleResponse(t *testing.T) {
	model := &mockModel{responses: []string{"Hello, I can help you."}}
	engine := New(WithModel(model), WithMaxTurns(5))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Run(ctx, "hi")
	require.NoError(t, err)
	assert.Equal(t, "Hello, I can help you.", result.Response)
	assert.Equal(t, 1, result.Turns)
	assert.Len(t, result.Messages, 2) // user + assistant
}

func TestEngine_Good_ToolCallAndResponse(t *testing.T) {
	model := &mockModel{responses: []string{
		"Let me check.\n```tool\n{\"name\": \"test_tool\", \"args\": {\"key\": \"val\"}}\n```\n",
		"The result was: tool output.",
	}}

	toolCalled := false
	tools := []Tool{{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			toolCalled = true
			assert.Equal(t, "val", args["key"])
			return "tool output", nil
		},
	}}

	engine := New(WithModel(model), WithTools(tools...), WithMaxTurns(5))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.Run(ctx, "do something")
	require.NoError(t, err)
	assert.True(t, toolCalled)
	assert.Equal(t, 2, result.Turns)
	assert.Contains(t, result.Response, "tool output")
}

func TestEngine_Bad_MaxTurnsExceeded(t *testing.T) {
	model := &mockModel{responses: []string{
		"```tool\n{\"name\": \"t\", \"args\": {}}\n```\n",
		"```tool\n{\"name\": \"t\", \"args\": {}}\n```\n",
		"```tool\n{\"name\": \"t\", \"args\": {}}\n```\n",
	}}

	tools := []Tool{{
		Name: "t", Description: "loop forever",
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return "ok", nil
		},
	}}

	engine := New(WithModel(model), WithTools(tools...), WithMaxTurns(2))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := engine.Run(ctx, "go")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max turns")
}

func TestEngine_Bad_ContextCancelled(t *testing.T) {
	model := &mockModel{responses: []string{"thinking..."}}
	engine := New(WithModel(model), WithMaxTurns(5))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := engine.Run(ctx, "hi")
	require.Error(t, err)
}
