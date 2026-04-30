// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestFlow_FlowRunStepOutput_Good(t *testing.T) {
	step := FlowRunStepOutput{Name: "build", Command: "task", Args: []string{"./..."}, Success: true}

	core.AssertEqual(t, "build", step.Name)
	core.AssertEqual(t, "task", step.Command)
	core.AssertEqual(t, []string{"./..."}, step.Args)
}

func TestFlow_flowStepDisplayName_Bad(t *testing.T) {
	step := flowDefinitionStep{Cmd: "task"}
	name := flowStepDisplayName(3, step)

	core.AssertEqual(t, "task", name)
	core.AssertContains(t, name, "task")
}

func TestFlow_flowStepOptions_Ugly(t *testing.T) {
	options := flowStepOptions([]string{"--dry-run", "--model=gpt-5.4", "plan-123"})

	dryRun := options.Get("dry-run")
	model := options.Get("model")
	arg := options.Get("_arg")
	core.RequireTrue(t, dryRun.OK)
	core.RequireTrue(t, model.OK)
	core.RequireTrue(t, arg.OK)
	core.AssertEqual(t, true, dryRun.Value)
	core.AssertEqual(t, "gpt-5.4", model.Value)
	core.AssertEqual(t, "plan-123", arg.Value)
}
