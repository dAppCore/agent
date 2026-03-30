// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	coremcp "forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DirectSubsystem talks to OpenBrain over HTTP without the IDE bridge.
//
//	sub := brain.NewDirect()
//	core.Println(sub.Name()) // "brain"
type DirectSubsystem struct {
	apiURL string
	apiKey string
}

var _ coremcp.Subsystem = (*DirectSubsystem)(nil)

// NewDirect builds the HTTP-backed OpenBrain subsystem.
//
//	sub := brain.NewDirect()
//	core.Println(sub.Name())
func NewDirect() *DirectSubsystem {
	apiURL := core.Env("CORE_BRAIN_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}

	apiKey := core.Env("CORE_BRAIN_KEY")
	keyPath := ""
	if apiKey == "" {
		keyPath = brainKeyPath(agentic.HomeDir())
		if keyPath != "" {
			if readResult := fs.Read(keyPath); readResult.OK {
				apiKey = core.Trim(readResult.Value.(string))
				if apiKey != "" {
					core.Info("brain direct subsystem loaded API key from file", "path", keyPath)
				}
			}
		}
	}
	if apiKey == "" {
		core.Warn("brain direct subsystem has no API key configured", "path", keyPath)
	}

	return &DirectSubsystem{
		apiURL: apiURL,
		apiKey: apiKey,
	}
}

// Name keeps the direct subsystem address stable for core.WithService and MCP.
//
//	name := sub.Name() // "brain"
func (s *DirectSubsystem) Name() string { return "brain" }

// RegisterTools publishes the direct `brain_*` and `agent_*` tools on an MCP server.
//
//	sub := brain.NewDirect()
//	sub.RegisterTools(server)
func (s *DirectSubsystem) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in OpenBrain. Types: fact, decision, observation, plan, convention, architecture, research, documentation, service, bug, pattern, context, procedure.",
	}, s.remember)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across OpenBrain memories. Returns memories ranked by similarity. Use agent_id 'cladius' for Cladius's memories.",
	}, s.recall)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_forget",
		Description: "Remove a memory from OpenBrain by ID.",
	}, s.forget)

	// Agent messaging — direct, chronological, not semantic
	s.RegisterMessagingTools(server)
}

// Shutdown satisfies the MCP subsystem lifecycle without extra cleanup.
//
//	_ = sub.Shutdown(context.Background())
func (s *DirectSubsystem) Shutdown(_ context.Context) error { return nil }

func brainKeyPath(home string) string {
	if home == "" {
		return ""
	}
	return core.JoinPath(core.TrimSuffix(home, "/"), ".claude", "brain.key")
}

func (s *DirectSubsystem) apiCall(ctx context.Context, method, path string, body any) core.Result {
	if s.apiKey == "" {
		return core.Result{
			Value: core.E("brain.apiCall", "no API key (set CORE_BRAIN_KEY or create ~/.claude/brain.key)", nil),
			OK:    false,
		}
	}

	requestURL := core.Concat(s.apiURL, path)
	var bodyStr string
	if body != nil {
		bodyStr = core.JSONMarshalString(body)
	}
	requestResult := agentic.HTTPDo(ctx, method, requestURL, bodyStr, s.apiKey, "Bearer")
	if !requestResult.OK {
		core.Error("brain API call failed", "method", method, "path", path)
		if err, ok := requestResult.Value.(error); ok {
			return core.Result{Value: core.E("brain.apiCall", "API call failed", err), OK: false}
		}
		if responseBody, ok := requestResult.Value.(string); ok && responseBody != "" {
			return core.Result{Value: core.E("brain.apiCall", core.Concat("API call failed: ", core.Trim(responseBody)), nil), OK: false}
		}
		return core.Result{Value: core.E("brain.apiCall", "API call failed", nil), OK: false}
	}

	var result map[string]any
	if parseResult := core.JSONUnmarshalString(requestResult.Value.(string), &result); !parseResult.OK {
		core.Error("brain API response parse failed", "method", method, "path", path)
		err, _ := parseResult.Value.(error)
		return core.Result{Value: core.E("brain.apiCall", "parse response", err), OK: false}
	}

	return core.Result{Value: result, OK: true}
}

func (s *DirectSubsystem) remember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	result := s.apiCall(ctx, "POST", "/v1/brain/remember", map[string]any{
		"content":    input.Content,
		"type":       input.Type,
		"tags":       input.Tags,
		"project":    input.Project,
		"confidence": input.Confidence,
		"supersedes": input.Supersedes,
		"expires_in": input.ExpiresIn,
		"agent_id":   agentic.AgentName(),
	})
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, RememberOutput{}, err
	}

	payload, _ := result.Value.(map[string]any)
	id, _ := payload["id"].(string)
	return nil, RememberOutput{
		Success:   true,
		MemoryID:  id,
		Timestamp: time.Now(),
	}, nil
}

func (s *DirectSubsystem) recall(ctx context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	body := map[string]any{
		"query": input.Query,
		"top_k": input.TopK,
	}
	// Only filter by agent_id if explicitly provided — shared brain by default
	if input.Filter.AgentID != "" {
		body["agent_id"] = input.Filter.AgentID
	}
	if input.Filter.Project != "" {
		body["project"] = input.Filter.Project
	}
	if input.Filter.Type != nil {
		body["type"] = input.Filter.Type
	}
	if input.Filter.MinConfidence != 0 {
		body["min_confidence"] = input.Filter.MinConfidence
	}
	if input.TopK == 0 {
		body["top_k"] = 10
	}

	result := s.apiCall(ctx, "POST", "/v1/brain/recall", body)
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, RecallOutput{}, err
	}

	var memories []Memory
	payload, _ := result.Value.(map[string]any)
	if mems, ok := payload["memories"].([]any); ok {
		for _, m := range mems {
			if mm, ok := m.(map[string]any); ok {
				mem := Memory{
					Content:   stringField(mm, "content"),
					Type:      stringField(mm, "type"),
					Project:   stringField(mm, "project"),
					AgentID:   stringField(mm, "agent_id"),
					CreatedAt: stringField(mm, "created_at"),
				}
				if id, ok := mm["id"].(string); ok {
					mem.ID = id
				}
				if score, ok := mm["score"].(float64); ok {
					mem.Confidence = score
				}
				if tags, ok := mm["tags"].([]any); ok {
					for _, tag := range tags {
						mem.Tags = append(mem.Tags, core.Sprint(tag))
					}
				}
				if source, ok := mm["source"].(string); ok {
					mem.Tags = append(mem.Tags, core.Concat("source:", source))
				}
				memories = append(memories, mem)
			}
		}
	}

	return nil, RecallOutput{
		Success:  true,
		Count:    len(memories),
		Memories: memories,
	}, nil
}

func (s *DirectSubsystem) forget(ctx context.Context, _ *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	result := s.apiCall(ctx, "DELETE", core.Concat("/v1/brain/forget/", input.ID), nil)
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, ForgetOutput{}, err
	}

	return nil, ForgetOutput{
		Success:   true,
		Forgotten: input.ID,
		Timestamp: time.Now(),
	}, nil
}
