// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//	input := brain.RememberInput{
//	    Content: "Use core.Env for system paths.",
//	    Type:    "convention",
//	}
type RememberInput struct {
	Content    string   `json:"content"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags,omitempty"`
	Project    string   `json:"project,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	Supersedes string   `json:"supersedes,omitempty"`
	ExpiresIn  int      `json:"expires_in,omitempty"`
}

//	output := brain.RememberOutput{
//	    Success:  true,
//	    MemoryID: "mem_123",
//	}
type RememberOutput struct {
	Success   bool      `json:"success"`
	MemoryID  string    `json:"memory_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

//	input := brain.RecallInput{
//	    Query: "core.Env conventions",
//	    TopK:  5,
//	}
type RecallInput struct {
	Query  string       `json:"query"`
	TopK   int          `json:"top_k,omitempty"`
	Filter RecallFilter `json:"filter,omitempty"`
}

//	filter := brain.RecallFilter{
//	    Project: "agent",
//	    Type:    "convention",
//	}
type RecallFilter struct {
	Project       string  `json:"project,omitempty"`
	Type          any     `json:"type,omitempty"`
	AgentID       string  `json:"agent_id,omitempty"`
	MinConfidence float64 `json:"min_confidence,omitempty"`
}

//	output := brain.RecallOutput{
//	    Success: true,
//	    Count:   1,
//	}
type RecallOutput struct {
	Success  bool     `json:"success"`
	Count    int      `json:"count"`
	Memories []Memory `json:"memories"`
}

//	memory := brain.Memory{
//	    ID:      "mem_123",
//	    Type:    "convention",
//	    Content: "Use core.Env for system paths.",
//	}
type Memory struct {
	ID              string   `json:"id"`
	WorkspaceID     string   `json:"workspace_id,omitempty"`
	AgentID         string   `json:"agent_id"`
	Type            string   `json:"type"`
	Content         string   `json:"content"`
	Tags            []string `json:"tags,omitempty"`
	Project         string   `json:"project,omitempty"`
	Source          string   `json:"source,omitempty"`
	Confidence      float64  `json:"confidence"`
	SupersedesID    string   `json:"supersedes_id,omitempty"`
	SupersedesCount int      `json:"supersedes_count,omitempty"`
	ExpiresAt       string   `json:"expires_at,omitempty"`
	DeletedAt       string   `json:"deleted_at,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// memory := brain.BrainMemory{ID: "mem_123", Type: "convention", Content: "Use core.Trim for clean input."}
type BrainMemory = Memory

//	input := brain.ForgetInput{
//	    ID:     "mem_123",
//	    Reason: "superseded",
//	}
type ForgetInput struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

//	output := brain.ForgetOutput{
//	    Success:   true,
//	    Forgotten: "mem_123",
//	}
type ForgetOutput struct {
	Success   bool      `json:"success"`
	Forgotten string    `json:"forgotten"`
	Timestamp time.Time `json:"timestamp"`
}

//	input := brain.ListInput{
//	    Project: "agent",
//	    Limit:   20,
//	}
type ListInput struct {
	Project string `json:"project,omitempty"`
	Type    string `json:"type,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

//	output := brain.ListOutput{
//	    Success: true,
//	    Count:   2,
//	}
type ListOutput struct {
	Success  bool     `json:"success"`
	Count    int      `json:"count"`
	Memories []Memory `json:"memories"`
}

func (s *Subsystem) registerBrainTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in the shared OpenBrain knowledge store. Persists decisions, observations, conventions, research, plans, bugs, or architecture knowledge for other agents.",
	}, s.brainRemember)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across the shared OpenBrain knowledge store. Returns memories ranked by similarity to your query, with optional filtering.",
	}, s.brainRecall)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_forget",
		Description: "Remove a memory from the shared OpenBrain knowledge store. Permanently deletes from both database and vector index.",
	}, s.brainForget)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_list",
		Description: "List memories in the shared OpenBrain knowledge store. Supports filtering by project, type, and agent. No vector search -- use brain_recall for semantic queries.",
	}, s.brainList)
}

func (s *Subsystem) brainRemember(_ context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	if s.bridge == nil {
		return nil, RememberOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_remember",
		Data: map[string]any{
			"content":    input.Content,
			"type":       input.Type,
			"tags":       input.Tags,
			"project":    input.Project,
			"confidence": input.Confidence,
			"supersedes": input.Supersedes,
			"expires_in": input.ExpiresIn,
		},
	})
	if err != nil {
		return nil, RememberOutput{}, core.E("brain.remember", "failed to send brain_remember", err)
	}

	return nil, RememberOutput{
		Success:   true,
		Timestamp: time.Now(),
	}, nil
}

func (s *Subsystem) brainRecall(_ context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	if s.bridge == nil {
		return nil, RecallOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_recall",
		Data: map[string]any{
			"query":  input.Query,
			"top_k":  input.TopK,
			"filter": input.Filter,
		},
	})
	if err != nil {
		return nil, RecallOutput{}, core.E("brain.recall", "failed to send brain_recall", err)
	}

	return nil, RecallOutput{
		Success:  true,
		Memories: []Memory{},
	}, nil
}

func (s *Subsystem) brainForget(_ context.Context, _ *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	if s.bridge == nil {
		return nil, ForgetOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_forget",
		Data: map[string]any{
			"id":     input.ID,
			"reason": input.Reason,
		},
	})
	if err != nil {
		return nil, ForgetOutput{}, core.E("brain.forget", "failed to send brain_forget", err)
	}

	return nil, ForgetOutput{
		Success:   true,
		Forgotten: input.ID,
		Timestamp: time.Now(),
	}, nil
}

func (s *Subsystem) brainList(_ context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	if s.bridge == nil {
		return nil, ListOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_list",
		Data: map[string]any{
			"project":  input.Project,
			"type":     input.Type,
			"agent_id": input.AgentID,
			"limit":    input.Limit,
		},
	})
	if err != nil {
		return nil, ListOutput{}, core.E("brain.list", "failed to send brain_list", err)
	}

	return nil, ListOutput{
		Success:  true,
		Memories: []Memory{},
	}, nil
}
