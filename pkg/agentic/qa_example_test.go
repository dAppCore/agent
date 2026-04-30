// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleQAFinding() {
	finding := QAFinding{Severity: "error", Message: "missing test"}
	core.Println(finding.Severity)
	// Output: error
}
