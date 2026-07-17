// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"

	core "dappco.re/go"
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

	remote := service.DetectGitRemote(core.MustCast[string]((&core.Fs{}).NewUnrestricted().TempDir("example")))
	core.Println(remote == "")
	// Output: true
}

func ExampleService_OnStartup() {
	core.Println((&Service{}).OnStartup(context.Background()).OK)
	// Output: true
}
