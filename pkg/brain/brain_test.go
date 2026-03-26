// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Nil bridge tests (headless mode) ---

func TestBrain_Remember_Bad(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainRemember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
	})
	require.Error(t, err)
}

func TestBrain_Recall_Bad(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainRecall(context.Background(), nil, RecallInput{
		Query: "how does scoring work?",
	})
	require.Error(t, err)
}

func TestBrain_Forget_Bad(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainForget(context.Background(), nil, ForgetInput{
		ID: "550e8400-e29b-41d4-a716-446655440000",
	})
	require.Error(t, err)
}

func TestBrain_List_Bad(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainList(context.Background(), nil, ListInput{
		Project: "eaas",
	})
	require.Error(t, err)
}

// --- Subsystem interface tests ---

func TestBrain_Name_Good(t *testing.T) {
	sub := New(nil)
	assert.Equal(t, "brain", sub.Name())
}

func TestBrain_Shutdown_Good(t *testing.T) {
	sub := New(nil)
	assert.NoError(t, sub.Shutdown(context.Background()))
}

// --- Struct round-trip tests ---

// roundTrip marshals v to JSON and unmarshals into dst, failing on error.
func roundTrip(t *testing.T, v any, dst any) {
	t.Helper()
	s := core.JSONMarshalString(v)
	require.True(t, core.JSONUnmarshalString(s, dst).OK)
}

func TestBrain_RememberInput_Good(t *testing.T) {
	in := RememberInput{
		Content:    "LEM scoring was blind to negative emotions",
		Type:       "bug",
		Tags:       []string{"scoring", "lem"},
		Project:    "eaas",
		Confidence: 0.95,
		Supersedes: "550e8400-e29b-41d4-a716-446655440000",
		ExpiresIn:  24,
	}
	var out RememberInput
	roundTrip(t, in, &out)
	assert.Equal(t, in.Content, out.Content)
	assert.Equal(t, in.Type, out.Type)
	assert.Equal(t, []string{"scoring", "lem"}, out.Tags)
	assert.Equal(t, 0.95, out.Confidence)
}

func TestBrain_RememberOutput_Good(t *testing.T) {
	in := RememberOutput{
		Success:   true,
		MemoryID:  "550e8400-e29b-41d4-a716-446655440000",
		Timestamp: time.Now().Truncate(time.Second),
	}
	var out RememberOutput
	roundTrip(t, in, &out)
	assert.True(t, out.Success)
	assert.Equal(t, in.MemoryID, out.MemoryID)
}

func TestBrain_RecallInput_Good(t *testing.T) {
	in := RecallInput{
		Query: "how does verdict classification work?",
		TopK:  5,
		Filter: RecallFilter{
			Project:       "eaas",
			MinConfidence: 0.5,
		},
	}
	var out RecallInput
	roundTrip(t, in, &out)
	assert.Equal(t, in.Query, out.Query)
	assert.Equal(t, 5, out.TopK)
	assert.Equal(t, "eaas", out.Filter.Project)
	assert.Equal(t, 0.5, out.Filter.MinConfidence)
}

func TestBrain_Memory_Good(t *testing.T) {
	in := Memory{
		ID:         "550e8400-e29b-41d4-a716-446655440000",
		AgentID:    "virgil",
		Type:       "decision",
		Content:    "Use Qdrant for vector search",
		Tags:       []string{"architecture", "openbrain"},
		Project:    "php-agentic",
		Confidence: 0.9,
		CreatedAt:  "2026-03-03T12:00:00+00:00",
		UpdatedAt:  "2026-03-03T12:00:00+00:00",
	}
	var out Memory
	roundTrip(t, in, &out)
	assert.Equal(t, in.ID, out.ID)
	assert.Equal(t, "virgil", out.AgentID)
	assert.Equal(t, "decision", out.Type)
}

func TestBrain_ForgetInput_Good(t *testing.T) {
	in := ForgetInput{
		ID:     "550e8400-e29b-41d4-a716-446655440000",
		Reason: "Superseded by new approach",
	}
	var out ForgetInput
	roundTrip(t, in, &out)
	assert.Equal(t, in.ID, out.ID)
	assert.Equal(t, in.Reason, out.Reason)
}

func TestBrain_ListInput_Good(t *testing.T) {
	in := ListInput{
		Project: "eaas",
		Type:    "decision",
		AgentID: "charon",
		Limit:   20,
	}
	var out ListInput
	roundTrip(t, in, &out)
	assert.Equal(t, in, out)
}

func TestBrain_ListOutput_Good(t *testing.T) {
	in := ListOutput{
		Success: true,
		Count:   2,
		Memories: []Memory{
			{ID: "id-1", AgentID: "virgil", Type: "decision", Content: "memory 1", Confidence: 0.9, CreatedAt: "2026-03-03T12:00:00+00:00", UpdatedAt: "2026-03-03T12:00:00+00:00"},
			{ID: "id-2", AgentID: "charon", Type: "bug", Content: "memory 2", Confidence: 0.8, CreatedAt: "2026-03-03T13:00:00+00:00", UpdatedAt: "2026-03-03T13:00:00+00:00"},
		},
	}
	var out ListOutput
	roundTrip(t, in, &out)
	assert.True(t, out.Success)
	assert.Equal(t, 2, out.Count)
	require.Len(t, out.Memories, 2)
}
