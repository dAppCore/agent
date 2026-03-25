// SPDX-License-Identifier: EUPL-1.2

// Process execution helpers — wraps go-process for testable command execution.
// All external command execution in the agentic package goes through these helpers.

package agentic

import (
	"context"
	"sync"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

var procOnce sync.Once

// ensureProcess lazily initialises the default process service.
func ensureProcess() {
	procOnce.Do(func() {
		if process.Default() == nil {
			c := core.New()
			svc, err := process.NewService(process.Options{})(c)
			if err == nil {
				if s, ok := svc.(*process.Service); ok {
					process.SetDefault(s)
				}
			}
		}
	})
}

// runCmd executes a command in a directory and returns (output, error).
// Uses go-process RunWithOptions for testability.
//
//	out, err := runCmd(ctx, repoDir, "git", "log", "--oneline", "-20")
func runCmd(ctx context.Context, dir string, command string, args ...string) (string, error) {
	ensureProcess()
	return process.RunWithOptions(ctx, process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     dir,
	})
}

// runCmdEnv executes a command with additional environment variables.
//
//	out, err := runCmdEnv(ctx, repoDir, []string{"GOWORK=off"}, "go", "test", "./...")
func runCmdEnv(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
	ensureProcess()
	return process.RunWithOptions(ctx, process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     dir,
		Env:     env,
	})
}

// runCmdOK executes a command and returns true if it exits 0.
//
//	if runCmdOK(ctx, repoDir, "go", "build", "./...") { ... }
func runCmdOK(ctx context.Context, dir string, command string, args ...string) bool {
	_, err := runCmd(ctx, dir, command, args...)
	return err == nil
}

// gitCmd runs a git command in the given directory.
//
//	out, err := gitCmd(ctx, repoDir, "log", "--oneline", "-20")
func gitCmd(ctx context.Context, dir string, args ...string) (string, error) {
	return runCmd(ctx, dir, "git", args...)
}

// gitCmdOK runs a git command and returns true if it exits 0.
//
//	if gitCmdOK(ctx, repoDir, "fetch", "origin", "main") { ... }
func gitCmdOK(ctx context.Context, dir string, args ...string) bool {
	return runCmdOK(ctx, dir, "git", args...)
}

// gitOutput runs a git command and returns trimmed stdout.
//
//	branch := gitOutput(ctx, repoDir, "rev-parse", "--abbrev-ref", "HEAD")
func gitOutput(ctx context.Context, dir string, args ...string) string {
	out, err := gitCmd(ctx, dir, args...)
	if err != nil {
		return ""
	}
	return core.Trim(out)
}
