// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func ExampleService_DetectGitRemote() {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}

	// Non-git dir returns empty
	remote := svc.DetectGitRemote((&core.Fs{}).NewUnrestricted().TempDir("example"))
	core.Println(remote == "")
	// Output: true
}
