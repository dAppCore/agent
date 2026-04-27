// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
)

type brainClientState struct {
	once    core.Once
	breaker *brainclient.CircuitBreaker
}

func (s *PrepSubsystem) brainCall(ctx context.Context, method, path, agentID string, body map[string]any) core.Result {
	client := s.brainClient(agentID)
	if client == nil {
		return core.Result{Value: core.E("agentic.brainCall", "brain client unavailable", nil), OK: false}
	}

	payload, err := client.Call(ctx, method, path, s.scopedBrainBody(body))
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	return core.Result{Value: payload, OK: true}
}

func (s *PrepSubsystem) brainClient(agentID string) *brainclient.Client {
	if s == nil {
		return nil
	}

	return brainclient.New(brainclient.Options{
		URL:            s.brainURL,
		Key:            s.brainKey,
		Org:            core.Trim(core.Env("CORE_BRAIN_ORG")),
		AgentID:        core.Trim(agentID),
		CircuitBreaker: s.brainCircuitBreaker(),
	})
}

func (s *PrepSubsystem) brainCircuitBreaker() *brainclient.CircuitBreaker {
	state := s.brainClientState()
	if state == nil {
		return nil
	}

	state.once.Do(func() {
		state.breaker = brainclient.NewCircuitBreaker(brainclient.CircuitBreakerOptions{})
	})
	return state.breaker
}

func (s *PrepSubsystem) brainClientState() *brainClientState {
	if s == nil {
		return nil
	}

	s.brainStateOnce.Do(func() {
		s.brainState = &brainClientState{}
	})
	return s.brainState
}

func (s *PrepSubsystem) scopedBrainBody(body map[string]any) map[string]any {
	if body == nil {
		return nil
	}

	if org := core.Trim(core.Env("CORE_BRAIN_ORG")); org != "" && !brainValuePresent(body["org"]) {
		body["org"] = org
	}

	return body
}

func brainPayloadMap(payload map[string]any) map[string]any {
	if data, ok := payload["data"].(map[string]any); ok && len(data) > 0 {
		return data
	}
	return payload
}

func brainValuePresent(value any) bool {
	if value == nil {
		return false
	}

	return core.Trim(core.Sprint(value)) != ""
}
