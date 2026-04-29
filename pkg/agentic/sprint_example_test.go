// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleSprint() {
	sprint := Sprint{Title: "May hardening", Status: "active"}
	core.Println(sprint.Title)
	// Output: May hardening
}
