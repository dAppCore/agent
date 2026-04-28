// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go"

func ExampleNewDirect_name() {
	sub := NewDirect()
	core.Println(sub.Name())
	// Output: brain
}

func ExampleRememberInput() {
	input := RememberInput{Content: "Core uses Result pattern", Type: "observation"}
	core.Println(input.Type)
	// Output: observation
}

func ExampleRecallInput() {
	input := RecallInput{Query: "how does Core handle errors", TopK: 5}
	core.Println(input.TopK)
	// Output: 5
}
