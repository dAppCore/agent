// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// subsystem := brain.NewDirect()
// core.Println(subsystem.Name()) // "brain"
type DirectSubsystem struct {
	*core.ServiceRuntime[DirectOptions]
	apiURL    string
	apiKey    string
	apiClient *brainclient.Client
}

var _ coremcp.Subsystem = (*DirectSubsystem)(nil)

// subsystem := brain.NewDirect()
// core.Println(subsystem.Name())
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
		apiURL:    apiURL,
		apiKey:    apiKey,
		apiClient: newBrainClient(apiURL, apiKey),
	}
}

// name := subsystem.Name() // "brain"
func (s *DirectSubsystem) Name() string { return "brain" }

// subsystem := brain.NewDirect()
// subsystem.RegisterTools(svc)
func (s *DirectSubsystem) RegisterTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in OpenBrain. Types: fact, decision, observation, plan, convention, architecture, research, documentation, service, bug, pattern, context, procedure.",
	}, s.remember)

	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across OpenBrain memories. Returns memories ranked by similarity. Use agent_id 'cladius' for Cladius's memories.",
	}, s.recall)

	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "brain_forget",
		Description: "Remove a memory from OpenBrain by ID.",
	}, s.forget)

	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "brain_list",
		Description: "List memories in OpenBrain with optional project, type, agent, and limit filters.",
	}, s.list)

	s.RegisterMessagingTools(svc)
}

// _ = subsystem.Shutdown(context.Background())
func (s *DirectSubsystem) Shutdown(_ context.Context) error { return nil }

func brainKeyPath(home string) string {
	if home == "" {
		return ""
	}
	return core.JoinPath(core.TrimSuffix(home, "/"), ".claude", "brain.key")
}

func (s *DirectSubsystem) apiCall(ctx context.Context, method, path string, body any) core.Result {
	result, err := s.client().Call(ctx, method, path, body)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: result, OK: true}
}

func (s *DirectSubsystem) remember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	org := directOrg(input.Org)
	result := s.apiCall(ctx, "POST", "/v1/brain/remember", map[string]any{
		"content":    input.Content,
		"type":       input.Type,
		"tags":       input.Tags,
		"org":        org,
		"project":    input.Project,
		"confidence": input.Confidence,
		"supersedes": input.Supersedes,
		"expires_in": input.ExpiresIn,
		"agent_id":   directAgentID(),
	})
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, RememberOutput{}, err
	}

	payload, _ := result.Value.(map[string]any)
	return nil, RememberOutput{
		Success:   true,
		MemoryID:  stringField(payloadMap(payload), "id"),
		Timestamp: time.Now(),
	}, nil
}

func (s *DirectSubsystem) recall(ctx context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	body := map[string]any{
		"query": input.Query,
		"top_k": input.TopK,
	}
	if input.Filter.AgentID != "" {
		body["agent_id"] = input.Filter.AgentID
	}
	if input.Filter.Project != "" {
		body["project"] = input.Filter.Project
	}
	if org := directOrg(input.Filter.Org); org != "" {
		body["org"] = org
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

	payload, _ := result.Value.(map[string]any)
	memories := memoriesFromPayload(payload)

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

func (s *DirectSubsystem) list(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	var params []string
	if org := directOrg(input.Org); org != "" {
		params = append(params, core.Concat("org=", core.URLEncode(org)))
	}
	if input.Project != "" {
		params = append(params, core.Concat("project=", core.URLEncode(input.Project)))
	}
	if input.Type != "" {
		params = append(params, core.Concat("type=", core.URLEncode(input.Type)))
	}
	if input.AgentID != "" {
		params = append(params, core.Concat("agent_id=", core.URLEncode(input.AgentID)))
	}
	if input.Limit > 0 {
		params = append(params, core.Concat("limit=", core.URLEncode(core.Sprint(input.Limit))))
	}

	path := "/v1/brain/list"
	if len(params) > 0 {
		path = core.Concat(path, "?", core.Join("&", params...))
	}

	result := s.apiCall(ctx, "GET", path, nil)
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, ListOutput{}, err
	}

	payload, _ := result.Value.(map[string]any)
	memories := memoriesFromPayload(payload)

	return nil, ListOutput{
		Success:  true,
		Count:    len(memories),
		Memories: memories,
	}, nil
}

func (s *DirectSubsystem) client() *brainclient.Client {
	if s.apiClient == nil {
		s.apiClient = newBrainClient(s.apiURL, s.apiKey)
	}
	return s.apiClient
}

func newBrainClient(apiURL, apiKey string) *brainclient.Client {
	return brainclient.New(brainclient.Options{
		URL:     apiURL,
		Key:     apiKey,
		Org:     core.Trim(core.Env("CORE_BRAIN_ORG")),
		AgentID: directAgentID(),
	})
}

func directAgentID() string {
	if configured := core.Trim(core.Env("CORE_BRAIN_AGENT_ID")); configured != "" {
		return configured
	}
	return agentic.AgentName()
}

func directOrg(org string) string {
	org = core.Trim(org)
	if org != "" {
		return org
	}
	return core.Trim(core.Env("CORE_BRAIN_ORG"))
}

func memoriesFromPayload(payload map[string]any) []Memory {
	var memories []Memory
	source := payloadMap(payload)
	if mems, ok := source["memories"].([]any); ok {
		for _, m := range mems {
			memoryMap, ok := m.(map[string]any)
			if !ok {
				continue
			}

			memory := Memory{
				Content:     stringField(memoryMap, "content"),
				Type:        stringField(memoryMap, "type"),
				Project:     stringField(memoryMap, "project"),
				WorkspaceID: stringField(memoryMap, "workspace_id"),
				AgentID:     stringField(memoryMap, "agent_id"),
				Source:      stringField(memoryMap, "source"),
				CreatedAt:   stringField(memoryMap, "created_at"),
				UpdatedAt:   stringField(memoryMap, "updated_at"),
				ExpiresAt:   stringField(memoryMap, "expires_at"),
				DeletedAt:   stringField(memoryMap, "deleted_at"),
			}
			if id, ok := memoryMap["id"].(string); ok {
				memory.ID = id
			}
			if score, ok := memoryMap["score"].(float64); ok {
				memory.Confidence = score
			}
			if confidence, ok := memoryMap["confidence"].(float64); ok && memory.Confidence == 0 {
				memory.Confidence = confidence
			}
			if supersedesID, ok := memoryMap["supersedes_id"].(string); ok {
				memory.SupersedesID = supersedesID
			}
			if supersedesCount, ok := memoryMap["supersedes_count"].(float64); ok {
				memory.SupersedesCount = int(supersedesCount)
			}
			if supersedesCount, ok := memoryMap["supersedes_count"].(int); ok {
				memory.SupersedesCount = supersedesCount
			}
			if tags, ok := memoryMap["tags"].([]any); ok {
				for _, tag := range tags {
					memory.Tags = append(memory.Tags, core.Sprint(tag))
				}
			}
			if source, ok := memoryMap["source"].(string); ok {
				if memory.Source == "" {
					memory.Source = source
				}
				memory.Tags = append(memory.Tags, core.Concat("source:", source))
			}
			memories = append(memories, memory)
		}
	}
	return memories
}

func payloadMap(payload map[string]any) map[string]any {
	if data, ok := payload["data"].(map[string]any); ok && len(data) > 0 {
		return data
	}
	return payload
}
