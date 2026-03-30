// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

func ExampleProcessAlive() {
	r := testCore.Process().Start(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"1"}},
		core.Option{Key: "detach", Value: true},
	))
	if !r.OK {
		return
	}
	proc := r.Value.(*process.Process)
	defer proc.Kill()

	core.Println(ProcessAlive(testCore, proc.ID, proc.Info().PID))
	// Output: true
}

func ExampleProcessTerminate() {
	r := testCore.Process().Start(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"1"}},
		core.Option{Key: "detach", Value: true},
	))
	if !r.OK {
		return
	}
	proc := r.Value.(*process.Process)
	defer proc.Kill()
	core.Println(ProcessTerminate(testCore, proc.ID, proc.Info().PID))
	// Output: true
}
