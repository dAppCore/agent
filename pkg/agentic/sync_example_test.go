// SPDX-License-Identifier: EUPL-1.2

package agentic

import "fmt"

func Example_shouldSyncStatus() {
	fmt.Println(shouldSyncStatus("completed"))
	fmt.Println(shouldSyncStatus("running"))
	// Output:
	// true
	// false
}
