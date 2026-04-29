// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestPlatformTools_SyncStatusInput_Good(t *testing.T) {
	input := SyncStatusInput{AgentID: "charon"}

	core.AssertEqual(t, "charon", input.AgentID)
	core.AssertContains(t, core.JSONMarshalString(input), `"agent_id":"charon"`)
}

func TestPlatformTools_FleetTaskAssignInput_Bad(t *testing.T) {
	input := FleetTaskAssignInput{Repo: "core/go-io", Task: "Fix tests"}
	text := core.JSONMarshalString(input)

	core.AssertContains(t, text, `"repo":"core/go-io"`)
	core.AssertContains(t, text, `"task":"Fix tests"`)
}

func TestPlatformTools_SubscriptionBudgetUpdateInput_Ugly(t *testing.T) {
	input := SubscriptionBudgetUpdateInput{
		AgentID: "charon",
		Limits:  map[string]any{"max_daily_hours": 2},
	}

	core.AssertEqual(t, "charon", input.AgentID)
	core.AssertEqual(t, 2, input.Limits["max_daily_hours"])
}
