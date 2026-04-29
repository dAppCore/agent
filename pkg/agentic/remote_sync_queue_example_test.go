// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleSyncRealClock_Now() {
	core.Println(!remoteSyncRealClock{}.Now().IsZero())
	// Output: true
}

func ExampleSyncRealClock_After() {
	<-remoteSyncRealClock{}.After(0)
	core.Println(true)
	// Output: true
}
