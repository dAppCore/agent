// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"time"

	core "dappco.re/go"
)

func ExampleNew() {
	mon := New(Options{Interval: 30 * time.Second})
	core.Println(mon.Name())
	// Output: monitor
}

func ExampleRegister() {
	c := core.New(core.WithService(Register))

	service := c.Service("monitor")
	svc, ok := service.Value.(*Subsystem)
	core.Println(ok)
	core.Println(svc.Name())
	// Output:
	// true
	// monitor
}
