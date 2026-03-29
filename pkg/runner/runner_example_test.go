// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	core "dappco.re/go/core"
)

func ExampleNew() {
	svc := New()
	core.Println(svc.Workspaces().Len())
	// Output: 0
}

func ExampleRegister() {
	c := core.New(core.WithOption("name", "runner-example"))
	r := Register(c)
	core.Println(r.OK)
	// Output: true
}
