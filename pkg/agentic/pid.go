// SPDX-License-Identifier: EUPL-1.2

package agentic

import "syscall"

// PIDAlive checks if an OS process is still alive via PID signal check.
//
//	if agentic.PIDAlive(pid) { ... }
func PIDAlive(pid int) bool {
	if pid > 0 {
		return syscall.Kill(pid, 0) == nil
	}
	return false
}

// PIDTerminate terminates a process via SIGTERM.
//
//	if agentic.PIDTerminate(pid) { ... }
func PIDTerminate(pid int) bool {
	if pid > 0 {
		return syscall.Kill(pid, syscall.SIGTERM) == nil
	}
	return false
}
