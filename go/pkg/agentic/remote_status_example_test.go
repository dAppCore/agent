// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExampleRemoteStatusOutput() {
	out := RemoteStatusOutput{Success: true}
	core.Println(out.Success)
	// Output: true
}
