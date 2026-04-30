// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	core "dappco.re/go"
)

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

func ExampleDirectSubsystem_Name() {
	core.Println(NewDirect().Name())
	// Output: brain
}

func ExampleDirectSubsystem_RegisterTools() {
	core.Println(brainExampleToolCount(NewDirect().RegisterTools) > 0)
	// Output: true
}

func ExampleDirectSubsystem_Shutdown() {
	core.Println(NewDirect().Shutdown(context.Background()) == nil)
	// Output: true
}
