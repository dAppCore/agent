// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DirectSubsystem calls the OpenBrain HTTP API without the IDE bridge.
//
//	sub := brain.NewDirect()
//	sub.RegisterTools(server)
type DirectSubsystem struct {
	apiURL string
	apiKey string
	client *http.Client
}

var _ coremcp.Subsystem = (*DirectSubsystem)(nil)

// NewDirect creates a direct HTTP brain subsystem.
//
//	sub := brain.NewDirect()
//	sub.RegisterTools(server)
func NewDirect() *DirectSubsystem {
	apiURL := core.Env("CORE_BRAIN_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}

	apiKey := core.Env("CORE_BRAIN_KEY")
	keyPath := ""
	if apiKey == "" {
		keyPath = brainKeyPath(brainHomeDir())
		if keyPath != "" {
			if r := fs.Read(keyPath); r.OK {
				apiKey = core.Trim(r.Value.(string))
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
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the MCP subsystem name.
//
//	name := sub.Name() // "brain"
func (s *DirectSubsystem) Name() string { return "brain" }

// RegisterTools adds the direct OpenBrain tools to an MCP server.
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

// Shutdown closes the direct subsystem without additional cleanup.
//
//	_ = sub.Shutdown(context.Background())
func (s *DirectSubsystem) Shutdown(_ context.Context) error { return nil }

func brainKeyPath(home string) string {
	if home == "" {
		return ""
	}
	return core.JoinPath(core.TrimSuffix(home, "/"), ".claude", "brain.key")
}

func brainHomeDir() string {
	if home := core.Env("CORE_HOME"); home != "" {
		return home
	}
	return core.Env("DIR_HOME")
}

func (s *DirectSubsystem) apiCall(ctx context.Context, method, path string, body any) (map[string]any, error) {
	if s.apiKey == "" {
		return nil, core.E("brain.apiCall", "no API key (set CORE_BRAIN_KEY or create ~/.claude/brain.key)", nil)
	}

	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			core.Error("brain API request marshal failed", "method", method, "path", path, "err", err)
			return nil, core.E("brain.apiCall", "marshal request", err)
		}
		reqBody = bytes.NewReader(data)
	}

	requestURL := core.Concat(s.apiURL, path)
	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	}
	if err != nil {
		core.Error("brain API request creation failed", "method", method, "path", path, "err", err)
		return nil, core.E("brain.apiCall", "create request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", core.Concat("Bearer ", s.apiKey))

	resp, err := s.client.Do(req)
	if err != nil {
		core.Error("brain API call failed", "method", method, "path", path, "err", err)
		return nil, core.E("brain.apiCall", "API call failed", err)
	}
	defer resp.Body.Close()

	respBuffer := bytes.NewBuffer(nil)
	if _, err := respBuffer.ReadFrom(resp.Body); err != nil {
		core.Error("brain API response read failed", "method", method, "path", path, "err", err)
		return nil, core.E("brain.apiCall", "read response", err)
	}
	respData := respBuffer.Bytes()

	if resp.StatusCode >= 400 {
		core.Warn("brain API returned error status", "method", method, "path", path, "status", resp.StatusCode)
		return nil, core.E("brain.apiCall", core.Sprintf("API returned %d: %s", resp.StatusCode, string(respData)), nil)
	}

	var result map[string]any
	if err := json.Unmarshal(respData, &result); err != nil {
		core.Error("brain API response parse failed", "method", method, "path", path, "err", err)
		return nil, core.E("brain.apiCall", "parse response", err)
	}

	return result, nil
}

func (s *DirectSubsystem) remember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	result, err := s.apiCall(ctx, "POST", "/v1/brain/remember", map[string]any{
		"content":    input.Content,
		"type":       input.Type,
		"tags":       input.Tags,
		"project":    input.Project,
		"confidence": input.Confidence,
		"supersedes": input.Supersedes,
		"expires_in": input.ExpiresIn,
		"agent_id":   agentic.AgentName(),
	})
	if err != nil {
		return nil, RememberOutput{}, err
	}

	id, _ := result["id"].(string)
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

	result, err := s.apiCall(ctx, "POST", "/v1/brain/recall", body)
	if err != nil {
		return nil, RecallOutput{}, err
	}

	var memories []Memory
	if mems, ok := result["memories"].([]any); ok {
		for _, m := range mems {
			if mm, ok := m.(map[string]any); ok {
				mem := Memory{
					Content:   fieldString(mm, "content"),
					Type:      fieldString(mm, "type"),
					Project:   fieldString(mm, "project"),
					AgentID:   fieldString(mm, "agent_id"),
					CreatedAt: fieldString(mm, "created_at"),
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
	_, err := s.apiCall(ctx, "DELETE", core.Concat("/v1/brain/forget/", input.ID), nil)
	if err != nil {
		return nil, ForgetOutput{}, err
	}

	return nil, ForgetOutput{
		Success:   true,
		Forgotten: input.ID,
		Timestamp: time.Now(),
	}, nil
}
