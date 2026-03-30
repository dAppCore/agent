// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"

	core "dappco.re/go/core"
)

func ExamplePrepSubsystem_cmdWorkspaceClean() {
	fsys := (&core.Fs{}).NewUnrestricted()
	root := fsys.TempDir("agentic-workspace-clean-example")
	defer fsys.DeleteAll(root)

	previous, hadPrevious := os.LookupEnv("CORE_WORKSPACE")
	_ = os.Setenv("CORE_WORKSPACE", root)
	defer func() {
		if hadPrevious {
			_ = os.Setenv("CORE_WORKSPACE", previous)
			return
		}
		_ = os.Unsetenv("CORE_WORKSPACE")
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
