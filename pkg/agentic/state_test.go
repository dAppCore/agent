// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_HandleStateSet_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: `{"name":"observer"}`},
		core.Option{Key: "type", Value: "general"},
		core.Option{Key: "description", Value: "Shared across sessions"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateOutput)
	require.True(t, ok)
	assert.Equal(t, "pattern", output.State.Key)
	assert.Equal(t, "general", output.State.Type)
	assert.Equal(t, "Shared across sessions", output.State.Description)

	states, err := readPlanStates("ax-follow-up")
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "observer", anyMapValue(states[0].Value)["name"])
	assert.Equal(t, "general", states[0].Type)
	assert.Equal(t, "Shared across sessions", states[0].Description)
}

func TestState_HandleStateSet_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	assert.False(t, result.OK)
}

func TestState_HandleStateSet_Ugly_Upsert(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	first := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: "observer"},
	))
	require.True(t, first.OK)

	second := subsystem.handleStateSet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: "pipeline"},
	))
	require.True(t, second.OK)

	states, err := readPlanStates("ax-follow-up")
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "pipeline", stringValue(states[0].Value))
}

func TestState_HandleStateGet_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writePlanStates("ax-follow-up", []WorkspaceState{{
		Key:         "pattern",
		Value:       "observer",
		Type:        "general",
		Description: "Shared across sessions",
	}}))

	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateOutput)
	require.True(t, ok)
	assert.Equal(t, "observer", stringValue(output.State.Value))
	assert.Equal(t, "general", output.State.Type)
	assert.Equal(t, "Shared across sessions", output.State.Description)
}

func TestState_HandleStateGet_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	assert.False(t, result.OK)
}

func TestState_HandleStateGet_Ugly_CorruptStateFile(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.True(t, fs.EnsureDir(stateRoot()).OK)
	require.True(t, fs.Write(statePath("ax-follow-up"), `[{broken`).OK)

	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	assert.False(t, result.OK)
}

func TestState_HandleStateList_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writePlanStates("ax-follow-up", []WorkspaceState{
		{Key: "pattern", Value: "observer", Type: "general"},
		{Key: "risk", Value: "auth", Type: "security"},
	}))

	result := subsystem.handleStateList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "type", Value: "security"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Total)
	require.Len(t, output.States, 1)
	assert.Equal(t, "risk", output.States[0].Key)
	assert.Equal(t, "security", output.States[0].Type)
}

func TestState_HandleStateList_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateList(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
}

func TestState_HandleStateList_Ugly_CorruptStateFile(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.True(t, fs.EnsureDir(stateRoot()).OK)
	require.True(t, fs.Write(statePath("ax-follow-up"), `{broken`).OK)

	result := subsystem.handleStateList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	assert.False(t, result.OK)
}

func TestState_HandleStateDelete_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writePlanStates("ax-follow-up", []WorkspaceState{
		{Key: "pattern", Value: "observer", Type: "general", Description: "Shared across sessions"},
		{Key: "risk", Value: "auth", Type: "security"},
	}))

	result := subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateDeleteOutput)
	require.True(t, ok)
	assert.Equal(t, "pattern", output.Deleted.Key)
	assert.Equal(t, "general", output.Deleted.Type)
	assert.Equal(t, "Shared across sessions", output.Deleted.Description)

	states, err := readPlanStates("ax-follow-up")
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "risk", states[0].Key)

	require.True(t, subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "risk"},
	)).OK)
	assert.False(t, fs.Exists(statePath("ax-follow-up")))
}

func TestState_HandleStateDelete_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
	))
	assert.False(t, result.OK)
}

func TestState_HandleStateDelete_Ugly_CorruptStateFile(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.True(t, fs.EnsureDir(stateRoot()).OK)
	require.True(t, fs.Write(statePath("ax-follow-up"), `{broken`).OK)

	result := subsystem.handleStateDelete(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	assert.False(t, result.OK)
}
