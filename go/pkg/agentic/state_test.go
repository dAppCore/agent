// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestState_HandleStateSet_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: `{"name":"observer"}`},
		core.Option{Key: "type", Value: "general"},
		core.Option{Key: "description", Value: "Shared across sessions"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(StateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pattern", output.State.Key)
	core.AssertEqual(t, "general", output.State.Type)
	core.AssertEqual(t, "Shared across sessions", output.State.Description)

	stateResult := readPlanStates("ax-follow-up")
	core.RequireTrue(t, stateResult.OK)
	states := stateResult.Value.([]WorkspaceState)
	core.AssertLen(t, states, 1)
	core.AssertEqual(t, "observer", anyMapValue(states[0].Value)["name"])
	core.AssertEqual(t, "general", states[0].Type)
	core.AssertEqual(t, "Shared across sessions", states[0].Description)
}

func TestState_HandleStateSet_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	core.AssertFalse(t, result.OK)
}

func TestState_HandleStateSet_Ugly_Upsert(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	first := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: "observer"},
	))
	core.RequireTrue(t, first.OK)

	second := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: "pipeline"},
	))
	core.RequireTrue(t, second.OK)

	stateResult := readPlanStates("ax-follow-up")
	core.RequireTrue(t, stateResult.OK)
	states := stateResult.Value.([]WorkspaceState)
	core.AssertLen(t, states, 1)
	core.AssertEqual(t, "pipeline", stringValue(states[0].Value))
}

func TestState_HandleStateGet_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireTrue(t, writePlanStates("ax-follow-up", []WorkspaceState{{
		Key:         "pattern",
		Value:       "observer",
		Type:        "general",
		Description: "Shared across sessions",
	}}).OK)

	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(StateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "observer", stringValue(output.State.Value))
	core.AssertEqual(t, "general", output.State.Type)
	core.AssertEqual(t, "Shared across sessions", output.State.Description)
}

func TestState_HandleStateGet_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	core.AssertFalse(t, result.OK)
}

func TestState_HandleStateGet_Ugly_CorruptStateFile(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireTrue(t, fs.EnsureDir(stateRoot()).OK)
	core.RequireTrue(t, fs.Write(statePath("ax-follow-up"), `[{broken`).OK)

	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	core.AssertFalse(t, result.OK)
}

func TestState_HandleStateList_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireTrue(t, writePlanStates("ax-follow-up", []WorkspaceState{
		{Key: "pattern", Value: "observer", Type: "general"},
		{Key: "risk", Value: "auth", Type: "security"},
	}).OK)

	result := subsystem.handleStateList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "type", Value: "security"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(StateListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Total)
	core.AssertLen(t, output.States, 1)
	core.AssertEqual(t, "risk", output.States[0].Key)
	core.AssertEqual(t, "security", output.States[0].Type)
}

func TestState_HandleStateList_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateList(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestState_HandleStateList_Ugly_CorruptStateFile(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireTrue(t, fs.EnsureDir(stateRoot()).OK)
	core.RequireTrue(t, fs.Write(statePath("ax-follow-up"), `{broken`).OK)

	result := subsystem.handleStateList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	core.AssertFalse(t, result.OK)
}

func TestState_HandleStateDelete_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireTrue(t, writePlanStates("ax-follow-up", []WorkspaceState{
		{Key: "pattern", Value: "observer", Type: "general", Description: "Shared across sessions"},
		{Key: "risk", Value: "auth", Type: "security"},
	}).OK)

	result := subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(StateDeleteOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pattern", output.Deleted.Key)
	core.AssertEqual(t, "general", output.Deleted.Type)
	core.AssertEqual(t, "Shared across sessions", output.Deleted.Description)

	stateResult := readPlanStates("ax-follow-up")
	core.RequireTrue(t, stateResult.OK)
	states := stateResult.Value.([]WorkspaceState)
	core.AssertLen(t, states, 1)
	core.AssertEqual(t, "risk", states[0].Key)

	core.RequireTrue(t, subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "risk"},
	)).OK)
	core.AssertFalse(t, fs.Exists(statePath("ax-follow-up")))
}

func TestState_HandleStateDelete_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	core.AssertFalse(t, result.OK)
}

func TestState_HandleStateDelete_Ugly_CorruptStateFile(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireTrue(t, fs.EnsureDir(stateRoot()).OK)
	core.RequireTrue(t, fs.Write(statePath("ax-follow-up"), `{broken`).OK)

	result := subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	core.AssertFalse(t, result.OK)
}
