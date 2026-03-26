// SPDX-License-Identifier: EUPL-1.2

// Process execution helpers — routes all commands through s.Core().Process().
// No direct os/exec or go-process imports.
//
// Requires go-process to be registered with Core via:
//
//	core.New(core.WithService(agentic.ProcessRegister))

package agentic

import (
	"context"
	"syscall"

	core "dappco.re/go/core"
)

// runCmd executes a command in a directory. Returns Result{Value: string, OK: bool}.
//
//	r := s.runCmd(ctx, repoDir, "git", "log", "--oneline", "-20")
//	if r.OK { output := r.Value.(string) }
func (s *PrepSubsystem) runCmd(ctx context.Context, dir string, command string, args ...string) core.Result {
	return s.Core().Process().RunIn(ctx, dir, command, args...)
}

// runCmdEnv executes a command with additional environment variables.
//
//	r := s.runCmdEnv(ctx, repoDir, []string{"GOWORK=off"}, "go", "test", "./...")
func (s *PrepSubsystem) runCmdEnv(ctx context.Context, dir string, env []string, command string, args ...string) core.Result {
	return s.Core().Process().RunWithEnv(ctx, dir, env, command, args...)
}

// runCmdOK executes a command and returns true if it exits 0.
//
//	if s.runCmdOK(ctx, repoDir, "go", "build", "./...") { ... }
func (s *PrepSubsystem) runCmdOK(ctx context.Context, dir string, command string, args ...string) bool {
	return s.runCmd(ctx, dir, command, args...).OK
}

// gitCmd runs a git command in the given directory.
//
//	r := s.gitCmd(ctx, repoDir, "log", "--oneline", "-20")
func (s *PrepSubsystem) gitCmd(ctx context.Context, dir string, args ...string) core.Result {
	return s.runCmd(ctx, dir, "git", args...)
}

// gitCmdOK runs a git command and returns true if it exits 0.
//
//	if s.gitCmdOK(ctx, repoDir, "fetch", "origin", "main") { ... }
func (s *PrepSubsystem) gitCmdOK(ctx context.Context, dir string, args ...string) bool {
	return s.gitCmd(ctx, dir, args...).OK
}

// gitOutput runs a git command and returns trimmed stdout.
//
//	branch := s.gitOutput(ctx, repoDir, "rev-parse", "--abbrev-ref", "HEAD")
func (s *PrepSubsystem) gitOutput(ctx context.Context, dir string, args ...string) string {
	r := s.gitCmd(ctx, dir, args...)
	if !r.OK {
		return ""
	}
	return core.Trim(r.Value.(string))
}

// --- Process lifecycle helpers ---

// processIsRunning checks if a process is still alive via PID signal check.
//
//	if processIsRunning(st.ProcessID, st.PID) { ... }
func processIsRunning(processID string, pid int) bool {
	if pid > 0 {
		return syscall.Kill(pid, 0) == nil
	}
	return false
}

// processKill terminates a process via SIGTERM.
//
//	processKill(st.ProcessID, st.PID)
func processKill(processID string, pid int) bool {
	if pid > 0 {
		return syscall.Kill(pid, syscall.SIGTERM) == nil
	}
	return false
}
