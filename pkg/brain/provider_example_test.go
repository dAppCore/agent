// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go"

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

func ExampleBrainProvider_Name() {
	core.Println(NewProvider(nil, nil).Name())
	// Output: brain
}

func ExampleBrainProvider_Element() {
	core.Println(NewProvider(nil, nil).Element().Tag)
	// Output: core-brain-panel
}

func ExampleBrainProvider_RegisterRoutes() {
	r := setupRouter(NewProvider(nil, nil))
	core.Println(len(providerRouteSignatures(r)))
	// Output: 5
}

func ExampleBrainProvider_Describe() {
	core.Println(len(NewProvider(nil, nil).Describe()))
	// Output: 5
}
