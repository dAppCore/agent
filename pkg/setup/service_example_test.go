// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func ExampleRegister_service() {
	c := core.New(core.WithService(Register))
	serviceResult := c.Service("setup")
	service, ok := serviceResult.Value.(*Service)
	core.Println(ok)
	core.Println(service != nil)
	// Output:
	// true
	// true
}

func ExampleService_DetectGitRemote() {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}

	// Non-git dir returns empty
	remote := service.DetectGitRemote((&core.Fs{}).NewUnrestricted().TempDir("example"))
	core.Println(remote == "")
	// Output: true
}
