// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func Example_parseSession() {
	session := parseSession(map[string]any{
		"session_id": "ses_abc123",
		"plan_slug":  "ax-follow-up",
		"agent_type": "codex",
		"status":     "active",
	})

	core.Println(session.SessionID, session.PlanSlug, session.AgentType, session.Status)
	// Output: ses_abc123 ax-follow-up codex active
}
