// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go/core"

func ExampleNewProvider() {
	p := NewProvider(nil, nil)
	core.Println(p.Name())
	// Output: brain
}

func ExampleBrainProvider_BasePath() {
	provider := NewProvider(nil, nil)
	core.Println(provider.BasePath())
	// Output: /api/brain
}

func ExampleBrainProvider_Channels() {
	provider := NewProvider(nil, nil)
	core.Println(len(provider.Channels()))
	// Output: 3
}
