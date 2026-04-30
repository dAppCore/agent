// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleIssue() {
	issue := Issue{Title: "Fix tests", Status: "open"}
	core.Println(issue.Title)
	// Output: Fix tests
}
