// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func Example_shouldSyncStatus() {
	core.Println(shouldSyncStatus("completed"))
	core.Println(shouldSyncStatus("running"))
	// Output:
	// true
	// false
}

func Example_syncBackoffSchedule() {
	core.Println(syncBackoffSchedule(1))
	core.Println(syncBackoffSchedule(2))
	core.Println(syncBackoffSchedule(3))
	core.Println(syncBackoffSchedule(4))
	core.Println(syncBackoffSchedule(5))
	// Output:
	// 1s
	// 5s
	// 15s
	// 1m0s
	// 5m0s
}
