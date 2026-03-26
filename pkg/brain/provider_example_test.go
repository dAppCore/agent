// SPDX-License-Identifier: EUPL-1.2

package brain

import core "dappco.re/go/core"

func ExampleNewDirect() {
	svc := NewDirect()
	core.Println(svc != nil)
	// Output: true
}

func ExampleNewProvider() {
	p := NewProvider(nil, nil)
	core.Println(p.Name())
	// Output: brain
}
