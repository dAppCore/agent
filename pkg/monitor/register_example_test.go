// SPDX-License-Identifier: EUPL-1.2

package monitor

import core "dappco.re/go/core"

func ExampleRegister_ipc() {
	c := core.New(core.WithService(Register))
	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	core.Println(ok)
	core.Println(svc.Name())
	// Output:
	// true
	// monitor
}
