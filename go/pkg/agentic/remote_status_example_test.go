// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleRemoteStatusOutput() {
	out := RemoteStatusOutput{Success: true}
	core.Println(out.Success)
	// Output: true
}
