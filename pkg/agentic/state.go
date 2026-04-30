// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// state := agentic.WorkspaceState{Key: "pattern", Value: "observer", Type: "general", Description: "Shared across sessions"}
type WorkspaceState struct {
	AgentPlanID int    `json:"agent_plan_id,omitempty"`
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// state := agentic.PlanState{Key: "pattern", Value: "observer", Type: "general"}
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
	return typedResultValue[StateOutput]("state.set", "invalid state set output", s.stateSet(ctx, StateSetInput{
		PlanSlug:    optionStringValue(options, "plan_slug", "plan"),
		Key:         optionStringValue(options, "key"),
		Value:       optionAnyValue(options, "value"),
		Type:        optionStringValue(options, "type"),
		Description: optionStringValue(options, "description"),
		Category:    optionStringValue(options, "category"),
	}))
}

// result := c.Action("state.get").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//	core.Option{Key: "key", Value: "pattern"},
//
// ))
func (s *PrepSubsystem) handleStateGet(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[StateOutput]("state.get", "invalid state get output", s.stateGet(ctx, StateGetInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Key:      optionStringValue(options, "key"),
	}))
}

// result := c.Action("state.list").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "ax-follow-up"}))
func (s *PrepSubsystem) handleStateList(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[StateListOutput]("state.list", "invalid state list output", s.stateList(ctx, StateListInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Type:     optionStringValue(options, "type"),
		Category: optionStringValue(options, "category"),
	}))
}

// result := c.Action("state.delete").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//	core.Option{Key: "key", Value: "pattern"},
//
// ))
func (s *PrepSubsystem) handleStateDelete(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[StateDeleteOutput]("state.delete", "invalid state delete output", s.stateDelete(ctx, StateDeleteInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Key:      optionStringValue(options, "key"),
	}))
}

func (s *PrepSubsystem) registerStateTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "state_set",
		Description: "Set a typed workspace state value for a plan so later sessions can reuse shared context.",
	}, toolHandlerFor[StateSetInput, StateOutput]("state.set", "invalid state set output", s.stateSet))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_state_set",
		Description: "Set a typed workspace state value for a plan so later sessions can reuse shared context.",
	}, toolHandlerFor[StateSetInput, StateOutput]("state.set", "invalid state set output", s.stateSet))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "state_get",
		Description: "Get a workspace state value for a plan by key.",
	}, toolHandlerFor[StateGetInput, StateOutput]("state.get", "invalid state get output", s.stateGet))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_state_get",
		Description: "Get a workspace state value for a plan by key.",
	}, toolHandlerFor[StateGetInput, StateOutput]("state.get", "invalid state get output", s.stateGet))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "state_list",
		Description: "List all stored workspace state values for a plan, with optional type or category filtering.",
	}, toolHandlerFor[StateListInput, StateListOutput]("state.list", "invalid state list output", s.stateList))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_state_list",
		Description: "List all stored workspace state values for a plan, with optional type or category filtering.",
	}, toolHandlerFor[StateListInput, StateListOutput]("state.list", "invalid state list output", s.stateList))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "state_delete",
		Description: "Delete a stored workspace state value for a plan by key.",
	}, toolHandlerFor[StateDeleteInput, StateDeleteOutput]("state.delete", "invalid state delete output", s.stateDelete))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_state_delete",
		Description: "Delete a stored workspace state value for a plan by key.",
	}, toolHandlerFor[StateDeleteInput, StateDeleteOutput]("state.delete", "invalid state delete output", s.stateDelete))
}

func (s *PrepSubsystem) stateSet(_ context.Context, input StateSetInput) core.Result {
	if input.PlanSlug == "" {
		return core.Fail(core.E("stateSet", "plan_slug is required", nil))
	}
	if input.Key == "" {
		return core.Fail(core.E("stateSet", "key is required", nil))
	}
	if input.Value == nil {
		return core.Fail(core.E("stateSet", "value is required", nil))
	}

	statesResult := readPlanStates(input.PlanSlug)
	if !statesResult.OK {
		return statesResult
	}
	states, _ := statesResult.Value.([]WorkspaceState)

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

	writeResult := writePlanStates(input.PlanSlug, states)
	if !writeResult.OK {
		return writeResult
	}

	return core.Ok(StateOutput{
		Success: true,
		State:   state,
	})
}

func (s *PrepSubsystem) stateGet(_ context.Context, input StateGetInput) core.Result {
	if input.PlanSlug == "" {
		return core.Fail(core.E("stateGet", "plan_slug is required", nil))
	}
	if input.Key == "" {
		return core.Fail(core.E("stateGet", "key is required", nil))
	}

	statesResult := readPlanStates(input.PlanSlug)
	if !statesResult.OK {
		return statesResult
	}
	states, _ := statesResult.Value.([]WorkspaceState)

	for _, state := range states {
		if state.Key == input.Key {
			state = normaliseWorkspaceState(state)
			return core.Ok(StateOutput{
				Success: true,
				State:   state,
			})
		}
	}

	return core.Fail(core.E("stateGet", core.Concat("state not found: ", input.Key), nil))
}

func (s *PrepSubsystem) stateList(_ context.Context, input StateListInput) core.Result {
	if input.PlanSlug == "" {
		return core.Fail(core.E("stateList", "plan_slug is required", nil))
	}

	statesResult := readPlanStates(input.PlanSlug)
	if !statesResult.OK {
		return statesResult
	}
	states, _ := statesResult.Value.([]WorkspaceState)

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

	return core.Ok(StateListOutput{
		Success: true,
		Total:   len(filtered),
		States:  filtered,
	})
}

func (s *PrepSubsystem) stateDelete(_ context.Context, input StateDeleteInput) core.Result {
	if input.PlanSlug == "" {
		return core.Fail(core.E("stateDelete", "plan_slug is required", nil))
	}
	if input.Key == "" {
		return core.Fail(core.E("stateDelete", "key is required", nil))
	}

	statesResult := readPlanStates(input.PlanSlug)
	if !statesResult.OK {
		return statesResult
	}
	states, _ := statesResult.Value.([]WorkspaceState)

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
		return core.Fail(core.E("stateDelete", core.Concat("state not found: ", input.Key), nil))
	}

	path := statePath(input.PlanSlug)
	if len(filtered) == 0 {
		if deleteResult := fs.Delete(path); !deleteResult.OK {
			err, _ := deleteResult.Value.(error)
			return core.Fail(core.E("stateDelete", "failed to delete empty state file", err))
		}
	} else if writeResult := writePlanStates(input.PlanSlug, filtered); !writeResult.OK {
		return writeResult
	}

	return core.Ok(StateDeleteOutput{
		Success: true,
		Deleted: deleted,
	})
}

func stateRoot() string {
	return core.JoinPath(CoreRoot(), "state")
}

func statePath(planSlug string) string {
	return core.JoinPath(stateRoot(), core.Concat(pathKey(planSlug), ".json"))
}

func readPlanStates(planSlug string) core.Result {
	result := fs.Read(statePath(planSlug))
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return core.Ok([]WorkspaceState{})
		}
		if core.Contains(err.Error(), "no such file") {
			return core.Ok([]WorkspaceState{})
		}
		return core.Fail(core.E("readPlanStates", "failed to read state file", err))
	}

	content := core.Trim(result.Value.(string))
	if content == "" {
		return core.Ok([]WorkspaceState{})
	}

	var states []WorkspaceState
	if parseResult := core.JSONUnmarshalString(content, &states); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return core.Fail(core.E("readPlanStates", "failed to parse state file", err))
	}

	return core.Ok(states)
}

func writePlanStates(planSlug string, states []WorkspaceState) core.Result {
	if ensureDirResult := fs.EnsureDir(stateRoot()); !ensureDirResult.OK {
		err, _ := ensureDirResult.Value.(error)
		return core.Fail(core.E("writePlanStates", "failed to create state directory", err))
	}

	if writeResult := fs.WriteAtomic(statePath(planSlug), core.JSONMarshalString(states)); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		return core.Fail(core.E("writePlanStates", "failed to write state file", err))
	}

	return core.Ok(nil)
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
