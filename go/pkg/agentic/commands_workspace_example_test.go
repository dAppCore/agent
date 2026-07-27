// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"syscall"

	core "dappco.re/go"
)

func ExamplePrepSubsystem_cmdWorkspaceClean() {
	fsys := (&core.Fs{}).NewUnrestricted()
	root := core.MustCast[string](fsys.TempDir("agentic-workspace-clean-example"))
	defer fsys.DeleteAll(root)

	previous, hadPrevious := syscall.Getenv("CORE_WORKSPACE")
	_ = syscall.Setenv("CORE_WORKSPACE", root)
	defer func() {
		if hadPrevious {
			_ = syscall.Setenv("CORE_WORKSPACE", previous)
			return
		}
		_ = syscall.Unsetenv("CORE_WORKSPACE")
	}()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
	}

	result := s.cmdWorkspaceClean(core.NewOptions())
	core.Println(result.OK)
	// Output:
	// nothing to clean
	// true
}
