// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleMirrorInput() {
	input := MirrorInput{Repo: "go-io"}
	core.Println(input.Repo)
	// Output: go-io
}
