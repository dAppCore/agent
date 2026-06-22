// SPDX-License-Identifier: EUPL-1.2

package main

import (
	core "dappco.re/go"

	"dappco.re/go/agent/pkg/agentic"
)

// shell drops the current terminal into an interactive shell inside a running
// container/VM: `core-agent shell <id> [--runtime <rt>] [--shell <path>]`. The
// runtime CLI owns the TTY, so this verb is a thin terminal hand-off to
// agentic.ContainerShell.
func (commands applicationCommandSet) shell(opts core.Options) core.Result {
	id := opts.String("_arg")
	if id == "" {
		applicationPrint("shell: <id> required (core-agent shell <container-id> [--runtime=<rt>] [--shell=<path>])")
		return core.Result{}
	}
	r := agentic.ContainerShell(agentic.ShellRequest{
		ID:      id,
		Runtime: opts.String("runtime"),
		Shell:   opts.String("shell"),
	})
	if !r.OK {
		applicationPrint("shell: %s", r.Error())
		return core.Result{}
	}
	return core.Result{OK: true}
}
