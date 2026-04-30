// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
	"dappco.re/go/process"
)

// alive := agentic.ProcessAlive(c, proc.ID, proc.Info().PID)
// alive := agentic.ProcessAlive(c, "", 12345)
func ProcessAlive(c *core.Core, processID string, pid int) bool {
	service, ok := lookupProcessService(c)
	if !ok {
		return false
	}

	if processID != "" {
		if result := service.Get(processID); result.OK {
			if proc, ok := result.Value.(*process.Process); ok {
				return proc.IsRunning()
			}
		}
	}

	if pid <= 0 {
		return false
	}

	for _, proc := range service.Running() {
		if proc.Info().PID == pid {
			return true
		}
	}

	return false
}

// terminated := agentic.ProcessTerminate(c, proc.ID, proc.Info().PID)
func ProcessTerminate(c *core.Core, processID string, pid int) bool {
	service, ok := lookupProcessService(c)
	if !ok {
		return false
	}

	if processID != "" {
		if result := service.Kill(processID); result.OK {
			return true
		}
	}

	if pid <= 0 {
		return false
	}

	for _, proc := range service.Running() {
		if proc.Info().PID == pid {
			return service.Kill(proc.ID).OK
		}
	}

	return false
}

func lookupProcessService(c *core.Core) (*process.Service, bool) {
	if c == nil {
		return nil, false
	}
	result := c.Service("process")
	if !result.OK {
		return nil, false
	}
	service, ok := result.Value.(*process.Service)
	return service, ok
}
