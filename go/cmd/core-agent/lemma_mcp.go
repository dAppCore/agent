// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/chathistory"
	"dappco.re/go/agent/pkg/lemma"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// lemmaSubsystem exposes the local Lemma model as the lemma_send MCP
// tool. Each call opens the caller agent's portable chathistory archive
// at ~/Lethean/data/users/<agent_id>/chats.duckdb, appends the user +
// assistant turns, and returns the reply. Pass conversation_id to
// continue a thread; empty starts fresh.
//
// Subsystem-level config is template-only — Config.History is set per
// call from the agent_id input so each caller's conversations stay in
// their own DuckDB file (continuity-rights: the file is the agent's
// property).
type lemmaSubsystem struct {
	cfg        lemma.Config
	historyDir string
}

var _ coremcp.Subsystem = (*lemmaSubsystem)(nil)

// newLemmaSubsystem reads LEMMA_BASE_URL / LEMMA_MODEL / LEMMA_HISTORY_DIR
// env vars and applies the package defaults otherwise.
//
//	sub := newLemmaSubsystem()
//	_ = sub.Name() // "lemma"
func newLemmaSubsystem() *lemmaSubsystem {
	baseURL := core.Env("LEMMA_BASE_URL")
	if baseURL == "" {
		baseURL = lemma.DefaultBaseURL
	}
	model := core.Env("LEMMA_MODEL")
	if model == "" {
		model = lemma.DefaultModelID
	}
	historyDir := core.Env("LEMMA_HISTORY_DIR")
	if historyDir == "" {
		historyDir = core.PathJoin(core.Env("HOME"), "Lethean", "data", "users")
	}
	return &lemmaSubsystem{
		cfg: lemma.Config{
			BaseURL: baseURL,
			ModelID: model,
		},
		historyDir: historyDir,
	}
}

// registerLemmaSubsystem is the core.WithService factory.
//
//	core.WithService(registerLemmaSubsystem)
func registerLemmaSubsystem(_ *core.Core) core.Result {
	return core.Ok(newLemmaSubsystem())
}

// Name returns the subsystem id under which lemma_send registers.
func (s *lemmaSubsystem) Name() string { return "lemma" }

// Shutdown is a no-op — the subsystem holds no long-lived resources;
// chathistory handles open + close per tool invocation.
func (s *lemmaSubsystem) Shutdown(_ context.Context) error { return nil }

// LemmaSendInput is the lemma_send tool's input shape.
type LemmaSendInput struct {
	AgentID        string `json:"agent_id"`
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id,omitempty"`
	Title          string `json:"title,omitempty"`
}

// LemmaSendOutput is the lemma_send tool's output shape. ConversationID
// is the load-bearing field for multi-turn continuation — capture it
// from the first call, pass it back on the next.
type LemmaSendOutput struct {
	Reply          string `json:"reply"`
	ConversationID string `json:"conversation_id"`
}

// RegisterTools wires the lemma_send tool into the MCP service.
//
//	sub.RegisterTools(svc)
func (s *lemmaSubsystem) RegisterTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "lemma", &mcp.Tool{
		Name:        "lemma_send",
		Description: "Send a message to the local Lemma model and get a reply. Auto-captures both turns into the caller agent's portable chathistory archive at ~/Lethean/data/users/<agent_id>/chats.duckdb (continuity-rights: the file is the agent's property). Pass conversation_id to continue a thread; leave empty to start fresh.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input LemmaSendInput) (*mcp.CallToolResult, LemmaSendOutput, error) {
		return s.handleSend(ctx, input)
	})
}

// handleSend opens the caller's chathistory, starts or resumes the
// conversation, sends the message, and returns the reply + conv id.
func (s *lemmaSubsystem) handleSend(ctx context.Context, input LemmaSendInput) (*mcp.CallToolResult, LemmaSendOutput, error) {
	if core.Trim(input.AgentID) == "" {
		return nil, LemmaSendOutput{}, core.E("lemma_send", "agent_id required", nil)
	}
	if core.Trim(input.Message) == "" {
		return nil, LemmaSendOutput{}, core.E("lemma_send", "message required", nil)
	}

	histPath := core.PathJoin(s.historyDir, input.AgentID, "chats.duckdb")
	hist, err := chathistory.Open(input.AgentID, histPath)
	if err != nil {
		return nil, LemmaSendOutput{}, err
	}
	defer func() {
		// Reported, not dropped: this handle is written to, so a failed
		// close can mean the last turns never reached the archive.
		if err := hist.Close(); err != nil {
			core.Warn("chat: failed to close history archive", "reason", err)
		}
	}()

	cfg := s.cfg
	cfg.History = hist
	svc := lemma.New(cfg)

	var session *lemma.Session
	if core.Trim(input.ConversationID) != "" {
		session = svc.Resume(input.AgentID, input.ConversationID)
	} else {
		session, err = svc.StartSession(input.AgentID, lemma.SessionMeta{
			Title: input.Title,
			Tags:  []string{"mcp:lemma_send"},
		})
		if err != nil {
			return nil, LemmaSendOutput{}, err
		}
	}

	reply, err := session.Send(ctx, input.Message)
	if err != nil {
		return nil, LemmaSendOutput{}, err
	}

	return nil, LemmaSendOutput{
		Reply:          reply,
		ConversationID: session.ConversationID(),
	}, nil
}
