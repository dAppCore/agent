// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func ExampleRegister_serviceFor() {
	c := core.New(core.WithService(Register))
	svc, ok := core.ServiceFor[*Service](c, "setup")
	core.Println(ok)
	core.Println(svc != nil)
	// Output:
	// true
	// true
}

func ExampleService_DetectGitRemote() {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}

	// Non-git dir returns empty
	remote := svc.DetectGitRemote((&core.Fs{}).NewUnrestricted().TempDir("example"))
	core.Println(remote == "")
	// Output: true
}
