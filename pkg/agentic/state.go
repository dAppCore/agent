// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// state := agentic.WorkspaceState{Key: "pattern", Value: "observer", Type: "general", Description: "Shared across sessions"}
type WorkspaceState struct {
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// PlanState is kept as a compatibility alias for older callers.
type PlanState = WorkspaceState

// input := agentic.StateSetInput{PlanSlug: "ax-follow-up", Key: "pattern", Value: "observer", Type: "general", Description: "Shared across sessions"}
type StateSetInput struct {
	PlanSlug    string `json:"plan_slug"`
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

// input := agentic.StateGetInput{PlanSlug: "ax-follow-up", Key: "pattern"}
type StateGetInput struct {
	PlanSlug string `json:"plan_slug"`
	Key      string `json:"key"`
}

// input := agentic.StateListInput{PlanSlug: "ax-follow-up", Type: "general"}
type StateListInput struct {
	PlanSlug string `json:"plan_slug"`
	Type     string `json:"type,omitempty"`
	Category string `json:"category,omitempty"`
}

// input := agentic.StateDeleteInput{PlanSlug: "ax-follow-up", Key: "pattern"}
type StateDeleteInput struct {
	PlanSlug string `json:"plan_slug"`
	Key      string `json:"key"`
}

// out := agentic.StateOutput{Success: true, State: agentic.WorkspaceState{Key: "pattern"}}
type StateOutput struct {
	Success bool           `json:"success"`
	State   WorkspaceState `json:"state"`
}

// out := agentic.StateListOutput{Success: true, Total: 1, States: []agentic.WorkspaceState{{Key: "pattern"}}}
type StateListOutput struct {
	Success bool             `json:"success"`
	Total   int              `json:"total"`
	States  []WorkspaceState `json:"states"`
}

// out := agentic.StateDeleteOutput{Success: true, Deleted: agentic.WorkspaceState{Key: "pattern"}}
type StateDeleteOutput struct {
	Success bool           `json:"success"`
	Deleted WorkspaceState `json:"deleted"`
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
		PlanSlug:    optionStringValue(options, "plan_slug", "plan"),
		Key:         optionStringValue(options, "key"),
		Value:       optionAnyValue(options, "value"),
		Type:        optionStringValue(options, "type"),
		Description: optionStringValue(options, "description"),
		Category:    optionStringValue(options, "category"),
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
		Type:     optionStringValue(options, "type"),
		Category: optionStringValue(options, "category"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("state.delete").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//	core.Option{Key: "key", Value: "pattern"},
//
// ))
func (s *PrepSubsystem) handleStateDelete(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.stateDelete(ctx, nil, StateDeleteInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Key:      optionStringValue(options, "key"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerStateTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_set",
		Description: "Set a typed workspace state value for a plan so later sessions can reuse shared context.",
	}, s.stateSet)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_state_set",
		Description: "Set a typed workspace state value for a plan so later sessions can reuse shared context.",
	}, s.stateSet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_get",
		Description: "Get a workspace state value for a plan by key.",
	}, s.stateGet)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_state_get",
		Description: "Get a workspace state value for a plan by key.",
	}, s.stateGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_list",
		Description: "List all stored workspace state values for a plan, with optional type or category filtering.",
	}, s.stateList)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_state_list",
		Description: "List all stored workspace state values for a plan, with optional type or category filtering.",
	}, s.stateList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "state_delete",
		Description: "Delete a stored workspace state value for a plan by key.",
	}, s.stateDelete)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_state_delete",
		Description: "Delete a stored workspace state value for a plan by key.",
	}, s.stateDelete)
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
	stateType := core.Trim(input.Type)
	if stateType == "" {
		stateType = core.Trim(input.Category)
	}
	if stateType == "" {
		stateType = "general"
	}
	state := WorkspaceState{
		Key:         input.Key,
		Value:       input.Value,
		Type:        stateType,
		Description: core.Trim(input.Description),
		Category:    stateType,
		UpdatedAt:   now,
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
			state = normaliseWorkspaceState(state)
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

	filtered := make([]WorkspaceState, 0, len(states))
	for _, state := range states {
		state = normaliseWorkspaceState(state)
		if input.Type != "" && state.Type != input.Type {
			continue
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

func (s *PrepSubsystem) stateDelete(_ context.Context, _ *mcp.CallToolRequest, input StateDeleteInput) (*mcp.CallToolResult, StateDeleteOutput, error) {
	if input.PlanSlug == "" {
		return nil, StateDeleteOutput{}, core.E("stateDelete", "plan_slug is required", nil)
	}
	if input.Key == "" {
		return nil, StateDeleteOutput{}, core.E("stateDelete", "key is required", nil)
	}

	states, err := readPlanStates(input.PlanSlug)
	if err != nil {
		return nil, StateDeleteOutput{}, err
	}

	filtered := make([]WorkspaceState, 0, len(states))
	deleted := WorkspaceState{}
	found := false
	for _, state := range states {
		if state.Key == input.Key {
			deleted = normaliseWorkspaceState(state)
			found = true
			continue
		}
		filtered = append(filtered, state)
	}

	if !found {
		return nil, StateDeleteOutput{}, core.E("stateDelete", core.Concat("state not found: ", input.Key), nil)
	}

	path := statePath(input.PlanSlug)
	if len(filtered) == 0 {
		if deleteResult := fs.Delete(path); !deleteResult.OK {
			err, _ := deleteResult.Value.(error)
			return nil, StateDeleteOutput{}, core.E("stateDelete", "failed to delete empty state file", err)
		}
	} else if err := writePlanStates(input.PlanSlug, filtered); err != nil {
		return nil, StateDeleteOutput{}, err
	}

	return nil, StateDeleteOutput{
		Success: true,
		Deleted: deleted,
	}, nil
}

func stateRoot() string {
	return core.JoinPath(CoreRoot(), "state")
}

func statePath(planSlug string) string {
	return core.JoinPath(stateRoot(), core.Concat(core.SanitisePath(planSlug), ".json"))
}

func readPlanStates(planSlug string) ([]WorkspaceState, error) {
	result := fs.Read(statePath(planSlug))
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return []WorkspaceState{}, nil
		}
		if core.Contains(err.Error(), "no such file") {
			return []WorkspaceState{}, nil
		}
		return nil, core.E("readPlanStates", "failed to read state file", err)
	}

	content := core.Trim(result.Value.(string))
	if content == "" {
		return []WorkspaceState{}, nil
	}

	var states []WorkspaceState
	if parseResult := core.JSONUnmarshalString(content, &states); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return nil, core.E("readPlanStates", "failed to parse state file", err)
	}

	return states, nil
}

func writePlanStates(planSlug string, states []WorkspaceState) error {
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

func normaliseWorkspaceState(state WorkspaceState) WorkspaceState {
	state.Type = core.Trim(state.Type)
	state.Category = core.Trim(state.Category)
	if state.Type == "" {
		state.Type = state.Category
	}
	if state.Category == "" {
		state.Category = state.Type
	}
	if state.Type == "" {
		state.Type = "general"
	}
	if state.Category == "" {
		state.Category = state.Type
	}
	state.Description = core.Trim(state.Description)
	return state
}
