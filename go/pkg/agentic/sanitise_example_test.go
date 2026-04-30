// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func Example_sanitiseBranchSlug() {
	core.Println(sanitiseBranchSlug("Fix the broken tests!", 40))
	// Output: fix-the-broken-tests
}

func Example_sanitiseBranchSlug_truncate() {
	core.Println(len(sanitiseBranchSlug("a very long task description that should be truncated to fit", 20)) <= 20)
	// Output: true
}
