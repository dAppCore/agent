// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
)

// --- Nil bridge tests (headless mode) ---

func TestBrain_Remember_Bad_Case(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainRemember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
	})
	core.AssertError(t, err)
}

func TestBrain_Recall_Bad_Case(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainRecall(context.Background(), nil, RecallInput{
		Query: "how does scoring work?",
	})
	core.AssertError(t, err)
}

func TestBrain_Forget_Bad_Case(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainForget(context.Background(), nil, ForgetInput{
		ID: "550e8400-e29b-41d4-a716-446655440000",
	})
	core.AssertError(t, err)
}

func TestBrain_List_Bad_Case(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainList(context.Background(), nil, ListInput{
		Project: "eaas",
	})
	core.AssertError(t, err)
}

// --- Subsystem interface tests ---

func TestBrain_Name_Good(t *testing.T) {
	sub := New(nil)
	got := sub.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestBrain_Shutdown_Good(t *testing.T) {
	sub := New(nil)
	err := sub.Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

// --- Struct round-trip tests ---

// roundTrip marshals v to JSON and unmarshals into dst, failing on error.
func roundTrip(t *testing.T, v any, dst any) {
	t.Helper()
	s := core.JSONMarshalString(v)
	core.RequireTrue(t, core.JSONUnmarshalString(s, dst).OK)
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
	core.AssertEqual(t, in.Content, out.Content)
	core.AssertEqual(t, in.Type, out.Type)
	core.AssertEqual(t, []string{"scoring", "lem"}, out.Tags)
	core.AssertEqual(t, 0.95, out.Confidence)
}

func TestBrain_RememberOutput_Good(t *testing.T) {
	in := RememberOutput{
		Success:   true,
		MemoryID:  "550e8400-e29b-41d4-a716-446655440000",
		Timestamp: time.Now().Truncate(time.Second),
	}
	var out RememberOutput
	roundTrip(t, in, &out)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, in.MemoryID, out.MemoryID)
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
	core.AssertEqual(t, in.Query, out.Query)
	core.AssertEqual(t, 5, out.TopK)
	core.AssertEqual(t, "eaas", out.Filter.Project)
	core.AssertEqual(t, 0.5, out.Filter.MinConfidence)
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
	core.AssertEqual(t, in.ID, out.ID)
	core.AssertEqual(t, "virgil", out.AgentID)
	core.AssertEqual(t, "decision", out.Type)
}

func TestBrain_ForgetInput_Good(t *testing.T) {
	in := ForgetInput{
		ID:     "550e8400-e29b-41d4-a716-446655440000",
		Reason: "Superseded by new approach",
	}
	var out ForgetInput
	roundTrip(t, in, &out)
	core.AssertEqual(t, in.ID, out.ID)
	core.AssertEqual(t, in.Reason, out.Reason)
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
	core.AssertEqual(t, in, out)
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
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 2, out.Count)
	core.AssertLen(t, out.Memories, 2)
}

func TestBrain_New_Good(t *testing.T) {
	bridge := ide.NewBridge(nil, ide.Config{})
	sub := New(bridge)
	core.AssertNotNil(t, sub)
	core.AssertSame(t, bridge, sub.bridge)
}

func TestBrain_New_Bad(t *testing.T) {
	sub := New(nil)
	core.AssertNotNil(t, sub)
	core.AssertNil(t, sub.bridge)
}

func TestBrain_New_Ugly(t *testing.T) {
	first := New(nil)
	second := New(nil)
	core.AssertTrue(t, first != second)
	core.AssertEqual(t, "brain", first.Name())
}

func TestBrain_Subsystem_Name_Good(t *testing.T) {
	got := New(nil).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestBrain_Subsystem_Name_Bad(t *testing.T) {
	got := (&Subsystem{}).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertContains(t, got, "brain")
}

func TestBrain_Subsystem_Name_Ugly(t *testing.T) {
	var sub *Subsystem
	got := sub.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotContains(t, got, "/")
}

func TestBrain_Subsystem_RegisterTools_Good(t *testing.T) {
	names := listedToolNames(t, New(nil).RegisterTools)
	core.AssertContains(t, names, "brain_remember")
	core.AssertContains(t, names, "brain_list")
}

func TestBrain_Subsystem_RegisterTools_Bad(t *testing.T) {
	names := listedToolNames(t, (&Subsystem{}).RegisterTools)
	core.AssertContains(t, names, "brain_recall")
	core.AssertContains(t, names, "brain_forget")
}

func TestBrain_Subsystem_RegisterTools_Ugly(t *testing.T) {
	names := listedToolNames(t, func(svc *coremcp.Service) {
		sub := New(nil)
		sub.RegisterTools(svc)
		sub.RegisterTools(svc)
	})
	core.AssertContains(t, names, "brain_remember")
	core.AssertContains(t, names, "brain_recall")
}

func TestBrain_Subsystem_Shutdown_Good(t *testing.T) {
	err := New(nil).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestBrain_Subsystem_Shutdown_Bad(t *testing.T) {
	err := (&Subsystem{}).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestBrain_Subsystem_Shutdown_Ugly(t *testing.T) {
	var sub *Subsystem
	core.AssertNotPanics(t, func() {
		core.AssertNoError(t, sub.Shutdown(context.Background()))
	})
}
