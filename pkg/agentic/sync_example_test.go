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

func Example_syncBackoffSchedule() {
	fmt.Println(syncBackoffSchedule(1))
	fmt.Println(syncBackoffSchedule(2))
	fmt.Println(syncBackoffSchedule(3))
	fmt.Println(syncBackoffSchedule(4))
	fmt.Println(syncBackoffSchedule(5))
	// Output:
	// 1s
	// 5s
	// 15s
	// 1m0s
	// 5m0s
}
