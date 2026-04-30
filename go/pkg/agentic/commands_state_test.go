// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestCommandsState_RegisterStateCommands_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerStateCommands()

	core.AssertContains(t, c.Commands(), "state")
	core.AssertContains(t, c.Commands(), "agentic:state")
	core.AssertContains(t, c.Commands(), "state/set")
	core.AssertContains(t, c.Commands(), "state/get")
	core.AssertContains(t, c.Commands(), "state/list")
	core.AssertContains(t, c.Commands(), "state/delete")
}

func TestCommandsState_CmdStateSet_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateSet(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
		core.Option{Key: "value", Value: "observer"},
		core.Option{Key: "type", Value: "general"},
		core.Option{Key: "description", Value: "Shared across sessions"},
	))

	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(StateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pattern", output.State.Key)
	core.AssertEqual(t, "general", output.State.Type)
	core.AssertEqual(t, "observer", output.State.Value)
	core.AssertEqual(t, "Shared across sessions", output.State.Description)
}

func TestCommandsState_CmdStateSet_Bad_MissingValue(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateSet(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))

	core.AssertFalse(t, result.OK)
}

func TestCommandsState_CmdStateGet_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	setResult := s.stateSet(s.commandContext(), StateSetInput{
		PlanSlug: "ax-follow-up",
		Key:      "pattern",
		Value:    "observer",
		Type:     "general",
	})
	core.RequireTrue(t, setResult.OK)
	output, ok := setResult.Value.(StateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pattern", output.State.Key)

	result := s.cmdStateGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))

	core.RequireTrue(t, result.OK)
	stateOutput, ok := result.Value.(StateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pattern", stateOutput.State.Key)
	core.AssertEqual(t, "observer", stateOutput.State.Value)
}

func TestCommandsState_CmdStateGet_Bad_MissingKey(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateGet(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	core.AssertFalse(t, result.OK)
}

func TestCommandsState_CmdStateList_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	setResult := s.stateSet(s.commandContext(), StateSetInput{
		PlanSlug: "ax-follow-up",
		Key:      "pattern",
		Value:    "observer",
		Type:     "general",
	})
	core.RequireTrue(t, setResult.OK)

	result := s.cmdStateList(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	core.RequireTrue(t, result.OK)
	listOutput, ok := result.Value.(StateListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, listOutput.Total)
	core.AssertLen(t, listOutput.States, 1)
}

func TestCommandsState_CmdStateList_Ugly_EmptyPlan(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateList(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	core.RequireTrue(t, result.OK)
	listOutput, ok := result.Value.(StateListOutput)
	core.RequireTrue(t, ok)
	assertZero(t, listOutput.Total)
	core.AssertEmpty(t, listOutput.States)
}

func TestCommandsState_CmdStateDelete_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	setResult := s.stateSet(s.commandContext(), StateSetInput{
		PlanSlug: "ax-follow-up",
		Key:      "pattern",
		Value:    "observer",
		Type:     "general",
	})
	core.RequireTrue(t, setResult.OK)

	result := s.cmdStateDelete(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "key", Value: "pattern"},
	))

	core.RequireTrue(t, result.OK)
	deleteOutput, ok := result.Value.(StateDeleteOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pattern", deleteOutput.Deleted.Key)
	core.AssertFalse(t, fs.Exists(statePath("ax-follow-up")))
}

func TestCommandsState_CmdStateDelete_Bad_MissingKey(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdStateDelete(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))

	core.AssertFalse(t, result.OK)
}
