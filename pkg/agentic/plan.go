// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Plan represents an implementation plan for agent work.
//
//	plan := &Plan{ID: "id-1-a3f2b1", Title: "Migrate Core", Status: "draft", Objective: "..."}
//	writePlan(PlansRoot(), plan)
type Plan struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // draft, ready, in_progress, needs_verification, verified, approved
	Repo      string    `json:"repo,omitempty"`
	Org       string    `json:"org,omitempty"`
	Objective string    `json:"objective"`
	Phases    []Phase   `json:"phases,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Phase represents a phase within an implementation plan.
//
//	phase := agentic.Phase{Number: 1, Name: "Migrate strings", Status: "in_progress"}
type Phase struct {
	Number   int      `json:"number"`
	Name     string   `json:"name"`
	Status   string   `json:"status"` // pending, in_progress, done
	Criteria []string `json:"criteria,omitempty"`
	Tests    int      `json:"tests,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

// --- Input/Output types ---

// PlanCreateInput is the input for agentic_plan_create.
//
//	input := agentic.PlanCreateInput{Title: "Migrate pkg/agentic", Objective: "Use Core primitives everywhere"}
type PlanCreateInput struct {
	Title     string  `json:"title"`
	Objective string  `json:"objective"`
	Repo      string  `json:"repo,omitempty"`
	Org       string  `json:"org,omitempty"`
	Phases    []Phase `json:"phases,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

// PlanCreateOutput is the output for agentic_plan_create.
//
//	out := agentic.PlanCreateOutput{Success: true, ID: "id-1-a3f2b1"}
type PlanCreateOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Path    string `json:"path"`
}

// PlanReadInput is the input for agentic_plan_read.
//
//	input := agentic.PlanReadInput{ID: "id-1-a3f2b1"}
type PlanReadInput struct {
	ID string `json:"id"`
}

// PlanReadOutput is the output for agentic_plan_read.
//
//	out := agentic.PlanReadOutput{Success: true, Plan: agentic.Plan{ID: "id-1-a3f2b1"}}
type PlanReadOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// PlanUpdateInput is the input for agentic_plan_update.
//
//	input := agentic.PlanUpdateInput{ID: "id-1-a3f2b1", Status: "verified"}
type PlanUpdateInput struct {
	ID        string  `json:"id"`
	Status    string  `json:"status,omitempty"`
	Title     string  `json:"title,omitempty"`
	Objective string  `json:"objective,omitempty"`
	Phases    []Phase `json:"phases,omitempty"`
	Notes     string  `json:"notes,omitempty"`
	Agent     string  `json:"agent,omitempty"`
}

// PlanUpdateOutput is the output for agentic_plan_update.
//
//	out := agentic.PlanUpdateOutput{Success: true, Plan: agentic.Plan{Status: "verified"}}
type PlanUpdateOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// PlanDeleteInput is the input for agentic_plan_delete.
//
//	input := agentic.PlanDeleteInput{ID: "id-1-a3f2b1"}
type PlanDeleteInput struct {
	ID string `json:"id"`
}

// PlanDeleteOutput is the output for agentic_plan_delete.
//
//	out := agentic.PlanDeleteOutput{Success: true, Deleted: "id-1-a3f2b1"}
type PlanDeleteOutput struct {
	Success bool   `json:"success"`
	Deleted string `json:"deleted"`
}

// PlanListInput is the input for agentic_plan_list.
//
//	input := agentic.PlanListInput{Repo: "go-io", Status: "ready"}
type PlanListInput struct {
	Status string `json:"status,omitempty"`
	Repo   string `json:"repo,omitempty"`
}

// PlanListOutput is the output for agentic_plan_list.
//
//	out := agentic.PlanListOutput{Success: true, Count: 2, Plans: []agentic.Plan{{ID: "id-1-a3f2b1"}}}
type PlanListOutput struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Plans   []Plan `json:"plans"`
}

// --- Registration ---

func (s *PrepSubsystem) registerPlanTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_create",
		Description: "Create an implementation plan. Plans track phased work with acceptance criteria, status lifecycle (draft → ready → in_progress → needs_verification → verified → approved), and per-phase progress.",
	}, s.planCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_read",
		Description: "Read an implementation plan by ID. Returns the full plan with all phases, criteria, and status.",
	}, s.planRead)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_update",
		Description: "Update an implementation plan. Supports partial updates — only provided fields are changed. Use this to advance status, update phases, or add notes.",
	}, s.planUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_delete",
		Description: "Delete an implementation plan by ID. Permanently removes the plan file.",
	}, s.planDelete)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_list",
		Description: "List implementation plans. Supports filtering by status (draft, ready, in_progress, etc.) and repo.",
	}, s.planList)
}

// --- Handlers ---

func (s *PrepSubsystem) planCreate(_ context.Context, _ *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, PlanCreateOutput, error) {
	if input.Title == "" {
		return nil, PlanCreateOutput{}, core.E("planCreate", "title is required", nil)
	}
	if input.Objective == "" {
		return nil, PlanCreateOutput{}, core.E("planCreate", "objective is required", nil)
	}

	id := generatePlanID(input.Title)
	plan := Plan{
		ID:        id,
		Title:     input.Title,
		Status:    "draft",
		Repo:      input.Repo,
		Org:       input.Org,
		Objective: input.Objective,
		Phases:    input.Phases,
		Notes:     input.Notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Default phase status to pending
	for i := range plan.Phases {
		if plan.Phases[i].Status == "" {
			plan.Phases[i].Status = "pending"
		}
		if plan.Phases[i].Number == 0 {
			plan.Phases[i].Number = i + 1
		}
	}

	writeResult := writePlanResult(PlansRoot(), &plan)
	if !writeResult.OK {
		err, _ := writeResult.Value.(error)
		if err == nil {
			err = core.E("planCreate", "failed to write plan", nil)
		}
		return nil, PlanCreateOutput{}, err
	}
	path, ok := writeResult.Value.(string)
	if !ok {
		return nil, PlanCreateOutput{}, core.E("planCreate", "invalid plan write result", nil)
	}

	return nil, PlanCreateOutput{
		Success: true,
		ID:      id,
		Path:    path,
	}, nil
}

func (s *PrepSubsystem) planRead(_ context.Context, _ *mcp.CallToolRequest, input PlanReadInput) (*mcp.CallToolResult, PlanReadOutput, error) {
	if input.ID == "" {
		return nil, PlanReadOutput{}, core.E("planRead", "id is required", nil)
	}

	planResult := readPlanResult(PlansRoot(), input.ID)
	if !planResult.OK {
		err, _ := planResult.Value.(error)
		if err == nil {
			err = core.E("planRead", "failed to read plan", nil)
		}
		return nil, PlanReadOutput{}, err
	}
	plan, ok := planResult.Value.(*Plan)
	if !ok || plan == nil {
		return nil, PlanReadOutput{}, core.E("planRead", "invalid plan payload", nil)
	}

	return nil, PlanReadOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

func (s *PrepSubsystem) planUpdate(_ context.Context, _ *mcp.CallToolRequest, input PlanUpdateInput) (*mcp.CallToolResult, PlanUpdateOutput, error) {
	if input.ID == "" {
		return nil, PlanUpdateOutput{}, core.E("planUpdate", "id is required", nil)
	}

	planResult := readPlanResult(PlansRoot(), input.ID)
	if !planResult.OK {
		err, _ := planResult.Value.(error)
		if err == nil {
			err = core.E("planUpdate", "failed to read plan", nil)
		}
		return nil, PlanUpdateOutput{}, err
	}
	plan, ok := planResult.Value.(*Plan)
	if !ok || plan == nil {
		return nil, PlanUpdateOutput{}, core.E("planUpdate", "invalid plan payload", nil)
	}

	// Apply partial updates
	if input.Status != "" {
		if !validPlanStatus(input.Status) {
			return nil, PlanUpdateOutput{}, core.E("planUpdate", core.Concat("invalid status: ", input.Status, " (valid: draft, ready, in_progress, needs_verification, verified, approved)"), nil)
		}
		plan.Status = input.Status
	}
	if input.Title != "" {
		plan.Title = input.Title
	}
	if input.Objective != "" {
		plan.Objective = input.Objective
	}
	if input.Phases != nil {
		plan.Phases = input.Phases
	}
	if input.Notes != "" {
		plan.Notes = input.Notes
	}
	if input.Agent != "" {
		plan.Agent = input.Agent
	}

	plan.UpdatedAt = time.Now()

	writeResult := writePlanResult(PlansRoot(), plan)
	if !writeResult.OK {
		err, _ := writeResult.Value.(error)
		if err == nil {
			err = core.E("planUpdate", "failed to write plan", nil)
		}
		return nil, PlanUpdateOutput{}, err
	}

	return nil, PlanUpdateOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

func (s *PrepSubsystem) planDelete(_ context.Context, _ *mcp.CallToolRequest, input PlanDeleteInput) (*mcp.CallToolResult, PlanDeleteOutput, error) {
	if input.ID == "" {
		return nil, PlanDeleteOutput{}, core.E("planDelete", "id is required", nil)
	}

	path := planPath(PlansRoot(), input.ID)
	if !fs.Exists(path) {
		return nil, PlanDeleteOutput{}, core.E("planDelete", core.Concat("plan not found: ", input.ID), nil)
	}

	if r := fs.Delete(path); !r.OK {
		err, _ := r.Value.(error)
		return nil, PlanDeleteOutput{}, core.E("planDelete", "failed to delete plan", err)
	}

	return nil, PlanDeleteOutput{
		Success: true,
		Deleted: input.ID,
	}, nil
}

func (s *PrepSubsystem) planList(_ context.Context, _ *mcp.CallToolRequest, input PlanListInput) (*mcp.CallToolResult, PlanListOutput, error) {
	dir := PlansRoot()
	if r := fs.EnsureDir(dir); !r.OK {
		err, _ := r.Value.(error)
		return nil, PlanListOutput{}, core.E("planList", "failed to access plans directory", err)
	}

	jsonFiles := core.PathGlob(core.JoinPath(dir, "*.json"))

	var plans []Plan
	for _, f := range jsonFiles {
		id := core.TrimSuffix(core.PathBase(f), ".json")
		planResult := readPlanResult(dir, id)
		if !planResult.OK {
			continue
		}
		plan, ok := planResult.Value.(*Plan)
		if !ok || plan == nil {
			continue
		}

		// Apply filters
		if input.Status != "" && plan.Status != input.Status {
			continue
		}
		if input.Repo != "" && plan.Repo != input.Repo {
			continue
		}

		plans = append(plans, *plan)
	}

	return nil, PlanListOutput{
		Success: true,
		Count:   len(plans),
		Plans:   plans,
	}, nil
}

// --- Helpers ---

func planPath(dir, id string) string {
	safe := core.SanitisePath(id)
	return core.JoinPath(dir, core.Concat(safe, ".json"))
}

// generatePlanID returns a Core ID for a plan file.
//
//	id := generatePlanID("Migrate Core") // "id-1-a3f2b1"
func generatePlanID(_ string) string {
	return core.ID()
}

// readPlanResult reads and decodes a plan file as core.Result.
//
//	result := readPlanResult(PlansRoot(), "plan-id")
//	if result.OK { plan := result.Value.(*Plan) }
func readPlanResult(dir, id string) core.Result {
	r := fs.Read(planPath(dir, id))
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("readPlan", core.Concat("plan not found: ", id), nil), OK: false}
		}
		return core.Result{Value: core.E("readPlan", core.Concat("plan not found: ", id), err), OK: false}
	}

	var plan Plan
	if ur := core.JSONUnmarshalString(r.Value.(string), &plan); !ur.OK {
		err, _ := ur.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("readPlan", core.Concat("failed to parse plan ", id), nil), OK: false}
		}
		return core.Result{Value: core.E("readPlan", core.Concat("failed to parse plan ", id), err), OK: false}
	}
	return core.Result{Value: &plan, OK: true}
}

// readPlan reads a plan file. Kept as compatibility wrapper.
//
//	plan, err := readPlan(PlansRoot(), "plan-id")
func readPlan(dir, id string) (*Plan, error) {
	r := readPlanResult(dir, id)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return nil, core.E("readPlan", "failed to read plan", nil)
		}
		return nil, err
	}
	plan, ok := r.Value.(*Plan)
	if !ok || plan == nil {
		return nil, core.E("readPlan", "invalid plan payload", nil)
	}
	return plan, nil
}

// writePlanResult writes a plan file and returns core.Result.
//
//	result := writePlanResult(PlansRoot(), plan)
//	if result.OK { path := result.Value.(string) }
func writePlanResult(dir string, plan *Plan) core.Result {
	if plan == nil {
		return core.Result{Value: core.E("writePlan", "plan is required", nil), OK: false}
	}
	if r := fs.EnsureDir(dir); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("writePlan", "failed to create plans directory", nil), OK: false}
		}
		return core.Result{Value: core.E("writePlan", "failed to create plans directory", err), OK: false}
	}

	path := planPath(dir, plan.ID)

	if r := fs.Write(path, core.JSONMarshalString(plan)); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("writePlan", "failed to write plan", nil), OK: false}
		}
		return core.Result{Value: core.E("writePlan", "failed to write plan", err), OK: false}
	}
	return core.Result{Value: path, OK: true}
}

// writePlan writes a plan file. Kept as compatibility wrapper.
//
//	_, err := writePlan(PlansRoot(), plan)
func writePlan(dir string, plan *Plan) (string, error) {
	r := writePlanResult(dir, plan)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return "", core.E("writePlan", "failed to write plan", nil)
		}
		return "", err
	}
	path, ok := r.Value.(string)
	if !ok {
		return "", core.E("writePlan", "invalid plan write result", nil)
	}
	return path, nil
}

func validPlanStatus(status string) bool {
	switch status {
	case "draft", "ready", "in_progress", "needs_verification", "verified", "approved":
		return true
	}
	return false
}
