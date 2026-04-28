// SPDX-License-Identifier: EUPL-1.2

package monitor

import core "dappco.re/go"

func ExampleRegister_ipc() {
	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	svc, ok := service.Value.(*Subsystem)
	core.Println(ok)
	core.Println(svc.Name())
	// Output:
	// true
	// monitor
}
