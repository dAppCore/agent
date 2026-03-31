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
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateOutput)
	require.True(t, ok)
	assert.Equal(t, "pattern", output.State.Key)

	states, err := readPlanStates("ax-follow-up")
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "observer", anyMapValue(states[0].Value)["name"])
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
	require.NoError(t, writePlanStates("ax-follow-up", []PlanState{{
		Key:      "pattern",
		Value:    "observer",
		Category: "general",
	}}))

	result := subsystem.handleStateGet(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateOutput)
	require.True(t, ok)
	assert.Equal(t, "observer", stringValue(output.State.Value))
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
	require.NoError(t, writePlanStates("ax-follow-up", []PlanState{
		{Key: "pattern", Value: "observer", Category: "general"},
		{Key: "risk", Value: "auth", Category: "security"},
	}))

	result := subsystem.handleStateList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "category", Value: "security"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(StateListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Total)
	require.Len(t, output.States, 1)
	assert.Equal(t, "risk", output.States[0].Key)
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
