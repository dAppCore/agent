// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// state := agentic.PlanState{Key: "pattern", Value: "observer", Category: "general"}
type PlanState struct {
	Key       string `json:"key"`
	Value     any    `json:"value"`
	Category  string `json:"category,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// input := agentic.StateSetInput{PlanSlug: "ax-follow-up", Key: "pattern", Value: "observer"}
type StateSetInput struct {
	PlanSlug string `json:"plan_slug"`
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Category string `json:"category,omitempty"`
}

// input := agentic.StateGetInput{PlanSlug: "ax-follow-up", Key: "pattern"}
type StateGetInput struct {
	PlanSlug string `json:"plan_slug"`
	Key      string `json:"key"`
}

// input := agentic.StateListInput{PlanSlug: "ax-follow-up", Category: "general"}
type StateListInput struct {
	PlanSlug string `json:"plan_slug"`
	Category string `json:"category,omitempty"`
}

// out := agentic.StateOutput{Success: true, State: agentic.PlanState{Key: "pattern"}}
type StateOutput struct {
	Success bool      `json:"success"`
	State   PlanState `json:"state"`
}

// out := agentic.StateListOutput{Success: true, Total: 1, States: []agentic.PlanState{{Key: "pattern"}}}
type StateListOutput struct {
	Success bool        `json:"success"`
	Total   int         `json:"total"`
	States  []PlanState `json:"states"`
}

// result := c.Action("state.set").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//	core.Option{Key: "key", Value: "pattern"},
//	core.Option{Key: "value", Value: "observer"},
//
// ))
func (s *PrepSubsystem) handleStateSet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.stateSet(ctx, nil, StateSetInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Key:      optionStringValue(options, "key"),
		Value:    optionAnyValue(options, "value"),
		Category: optionStringValue(options, "category"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("state.get").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//	core.Option{Key: "key", Value: "pattern"},
//
// ))
func (s *PrepSubsystem) handleStateGet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.stateGet(ctx, nil, StateGetInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Key:      optionStringValue(options, "key"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("state.list").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "ax-follow-up"}))
func (s *PrepSubsystem) handleStateList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.stateList(ctx, nil, StateListInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Category: optionStringValue(options, "category"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerStateTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_set",
		Description: "Set a workspace state value for a plan so later sessions can reuse shared context.",
	}, s.stateSet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_get",
		Description: "Get a workspace state value for a plan by key.",
	}, s.stateGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_list",
		Description: "List all stored workspace state values for a plan, with optional category filtering.",
	}, s.stateList)
}

func (s *PrepSubsystem) stateSet(_ context.Context, _ *mcp.CallToolRequest, input StateSetInput) (*mcp.CallToolResult, StateOutput, error) {
	if input.PlanSlug == "" {
		return nil, StateOutput{}, core.E("stateSet", "plan_slug is required", nil)
	}
	if input.Key == "" {
		return nil, StateOutput{}, core.E("stateSet", "key is required", nil)
	}
	if input.Value == nil {
		return nil, StateOutput{}, core.E("stateSet", "value is required", nil)
	}

	states, err := readPlanStates(input.PlanSlug)
	if err != nil {
		return nil, StateOutput{}, err
	}

	now := time.Now().Format(time.RFC3339)
	state := PlanState{
		Key:       input.Key,
		Value:     input.Value,
		Category:  input.Category,
		UpdatedAt: now,
	}
	if state.Category == "" {
		state.Category = "general"
	}

	found := false
	for i := range states {
		if states[i].Key == input.Key {
			states[i] = state
			found = true
			break
		}
	}
	if !found {
		states = append(states, state)
	}

	if err := writePlanStates(input.PlanSlug, states); err != nil {
		return nil, StateOutput{}, err
	}

	return nil, StateOutput{
		Success: true,
		State:   state,
	}, nil
}

func (s *PrepSubsystem) stateGet(_ context.Context, _ *mcp.CallToolRequest, input StateGetInput) (*mcp.CallToolResult, StateOutput, error) {
	if input.PlanSlug == "" {
		return nil, StateOutput{}, core.E("stateGet", "plan_slug is required", nil)
	}
	if input.Key == "" {
		return nil, StateOutput{}, core.E("stateGet", "key is required", nil)
	}

	states, err := readPlanStates(input.PlanSlug)
	if err != nil {
		return nil, StateOutput{}, err
	}

	for _, state := range states {
		if state.Key == input.Key {
			if state.Category == "" {
				state.Category = "general"
			}
			return nil, StateOutput{
				Success: true,
				State:   state,
			}, nil
		}
	}

	return nil, StateOutput{}, core.E("stateGet", core.Concat("state not found: ", input.Key), nil)
}

func (s *PrepSubsystem) stateList(_ context.Context, _ *mcp.CallToolRequest, input StateListInput) (*mcp.CallToolResult, StateListOutput, error) {
	if input.PlanSlug == "" {
		return nil, StateListOutput{}, core.E("stateList", "plan_slug is required", nil)
	}

	states, err := readPlanStates(input.PlanSlug)
	if err != nil {
		return nil, StateListOutput{}, err
	}

	filtered := make([]PlanState, 0, len(states))
	for _, state := range states {
		if state.Category == "" {
			state.Category = "general"
		}
		if input.Category != "" && state.Category != input.Category {
			continue
		}
		filtered = append(filtered, state)
	}

	return nil, StateListOutput{
		Success: true,
		Total:   len(filtered),
		States:  filtered,
	}, nil
}

func stateRoot() string {
	return core.JoinPath(CoreRoot(), "state")
}

func statePath(planSlug string) string {
	return core.JoinPath(stateRoot(), core.Concat(core.SanitisePath(planSlug), ".json"))
}

func readPlanStates(planSlug string) ([]PlanState, error) {
	result := fs.Read(statePath(planSlug))
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return []PlanState{}, nil
		}
		if core.Contains(err.Error(), "no such file") {
			return []PlanState{}, nil
		}
		return nil, core.E("readPlanStates", "failed to read state file", err)
	}

	content := core.Trim(result.Value.(string))
	if content == "" {
		return []PlanState{}, nil
	}

	var states []PlanState
	if parseResult := core.JSONUnmarshalString(content, &states); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return nil, core.E("readPlanStates", "failed to parse state file", err)
	}

	return states, nil
}

func writePlanStates(planSlug string, states []PlanState) error {
	if ensureDirResult := fs.EnsureDir(stateRoot()); !ensureDirResult.OK {
		err, _ := ensureDirResult.Value.(error)
		return core.E("writePlanStates", "failed to create state directory", err)
	}

	if writeResult := fs.WriteAtomic(statePath(planSlug), core.JSONMarshalString(states)); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		return core.E("writePlanStates", "failed to write state file", err)
	}

	return nil
}
