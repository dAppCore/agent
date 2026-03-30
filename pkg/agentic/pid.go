// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

// ProcessAlive checks whether a managed process is still running.
//
//	alive := agentic.ProcessAlive(c, proc.ID, proc.Info().PID)
//	alive := agentic.ProcessAlive(c, "", 12345) // legacy PID fallback
func ProcessAlive(c *core.Core, processID string, pid int) bool {
	if c == nil {
		return false
	}

	svc, ok := core.ServiceFor[*process.Service](c, "process")
	if !ok {
		return false
	}

	if processID != "" {
		if proc, err := svc.Get(processID); err == nil {
			return proc.IsRunning()
		}
	}

	if pid <= 0 {
		return false
	}

	for _, proc := range svc.Running() {
		if proc.Info().PID == pid {
			return true
		}
	}

	return false
}

// ProcessTerminate stops a managed process.
//
//	_ = agentic.ProcessTerminate(c, proc.ID, proc.Info().PID)
func ProcessTerminate(c *core.Core, processID string, pid int) bool {
	if c == nil {
		return false
	}

	svc, ok := core.ServiceFor[*process.Service](c, "process")
	if !ok {
		return false
	}

	if processID != "" {
		if err := svc.Kill(processID); err == nil {
			return true
		}
	}

	if pid <= 0 {
		return false
	}

	for _, proc := range svc.Running() {
		if proc.Info().PID == pid {
			return svc.Kill(proc.ID) == nil
		}
	}

	return false
}
