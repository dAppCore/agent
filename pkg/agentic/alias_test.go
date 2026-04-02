// SPDX-License-Identifier: EUPL-1.2

package agentic

import "testing"

func TestAlias_AgentPlan_Good(t *testing.T) {
	var plan AgentPlan
	plan.Title = "AX follow-up"
	plan.Status = "draft"

	if plan.Title != "AX follow-up" {
		t.Fatalf("expected AgentPlan alias to behave like Plan")
	}
	if plan.Status != "draft" {
		t.Fatalf("expected AgentPlan alias to behave like Plan")
	}
}

func TestAlias_AgentSession_Good(t *testing.T) {
	var session AgentSession
	session.SessionID = "ses-123"
	session.AgentType = "codex"

	if session.SessionID != "ses-123" {
		t.Fatalf("expected AgentSession alias to behave like Session")
	}
	if session.AgentType != "codex" {
		t.Fatalf("expected AgentSession alias to behave like Session")
	}
}
