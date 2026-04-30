// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleLanguageDetectInput() {
	input := LanguageDetectInput{Path: "pkg/agentic/prep.go"}
	core.Println(input.Path)
	// Output: pkg/agentic/prep.go
}
