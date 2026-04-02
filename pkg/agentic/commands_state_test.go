// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsState_RegisterStateCommands_Good(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerStateCommands()

	assert.Contains(t, c.Commands(), "state")
	assert.Contains(t, c.Commands(), "agentic:state")
	assert.Contains(t, c.Commands(), "state/set")
	assert.Contains(t, c.Commands(), "state/get")
	assert.Contains(t, c.Commands(), "state/list")
	assert.Contains(t, c.Commands(), "state/delete")
}

func TestCommandsState_CmdStateSet_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateSet(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: "observer"},
		core.Option{Key: "type", Value: "general"},
		core.Option{Key: "description", Value: "Shared across sessions"},
	))

	require.True(t, result.OK)

	output, ok := result.Value.(StateOutput)
	require.True(t, ok)
	assert.Equal(t, "pattern", output.State.Key)
	assert.Equal(t, "general", output.State.Type)
	assert.Equal(t, "observer", output.State.Value)
	assert.Equal(t, "Shared across sessions", output.State.Description)
}

func TestCommandsState_CmdStateSet_Bad_MissingValue(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateSet(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))

	assert.False(t, result.OK)
}

func TestCommandsState_CmdStateGet_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, output, err := s.stateSet(s.commandContext(), nil, StateSetInput{
		PlanSlug: "ax-follow-up",
		Key:      "pattern",
		Value:    "observer",
		Type:     "general",
	})
	require.NoError(t, err)
	require.Equal(t, "pattern", output.State.Key)

	result := s.cmdStateGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))

	require.True(t, result.OK)
	stateOutput, ok := result.Value.(StateOutput)
	require.True(t, ok)
	assert.Equal(t, "pattern", stateOutput.State.Key)
	assert.Equal(t, "observer", stateOutput.State.Value)
}

func TestCommandsState_CmdStateGet_Bad_MissingKey(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateGet(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	assert.False(t, result.OK)
}

func TestCommandsState_CmdStateList_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, _, err := s.stateSet(s.commandContext(), nil, StateSetInput{
		PlanSlug: "ax-follow-up",
		Key:      "pattern",
		Value:    "observer",
		Type:     "general",
	})
	require.NoError(t, err)

	result := s.cmdStateList(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	require.True(t, result.OK)
	listOutput, ok := result.Value.(StateListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, listOutput.Total)
	assert.Len(t, listOutput.States, 1)
}

func TestCommandsState_CmdStateList_Ugly_EmptyPlan(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateList(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	require.True(t, result.OK)
	listOutput, ok := result.Value.(StateListOutput)
	require.True(t, ok)
	assert.Zero(t, listOutput.Total)
	assert.Empty(t, listOutput.States)
}

func TestCommandsState_CmdStateDelete_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, _, err := s.stateSet(s.commandContext(), nil, StateSetInput{
		PlanSlug: "ax-follow-up",
		Key:      "pattern",
		Value:    "observer",
		Type:     "general",
	})
	require.NoError(t, err)

	result := s.cmdStateDelete(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))

	require.True(t, result.OK)
	deleteOutput, ok := result.Value.(StateDeleteOutput)
	require.True(t, ok)
	assert.Equal(t, "pattern", deleteOutput.Deleted.Key)
	assert.False(t, fs.Exists(statePath("ax-follow-up")))
}

func TestCommandsState_CmdStateDelete_Bad_MissingKey(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateDelete(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	assert.False(t, result.OK)
}
