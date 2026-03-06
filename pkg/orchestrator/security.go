package orchestrator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-\_\.]+$`)

// SanitizePath ensures a filename or directory name is safe and prevents path traversal.
// Returns filepath.Base of the input after validation.
func SanitizePath(input string) (string, error) {
	base := filepath.Base(input)
	if !safeNameRegex.MatchString(base) {
		return "", fmt.Errorf("agentci.SanitizePath: invalid characters in path element: %s", input)
	}
	if base == "." || base == ".." || base == "/" {
		return "", fmt.Errorf("agentci.SanitizePath: invalid path element: %s", base)
	}
	return base, nil
}

// EscapeShellArg wraps a string in single quotes for safe remote shell insertion.
// Prefer exec.Command arguments over constructing shell strings where possible.
func EscapeShellArg(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

// SecureSSHCommand creates an SSH exec.Cmd with strict host key checking and batch mode.
// Deprecated: Use SecureSSHCommandContext for context-aware cancellation.
func SecureSSHCommand(host string, remoteCmd string) *exec.Cmd {
	return SecureSSHCommandContext(context.Background(), host, remoteCmd)
}

// SecureSSHCommandContext creates an SSH exec.Cmd with context support for cancellation,
// strict host key checking, and batch mode.
func SecureSSHCommandContext(ctx context.Context, host string, remoteCmd string) *exec.Cmd {
	return exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		host,
		remoteCmd,
	)
}

// MaskToken returns a masked version of a token for safe logging.
func MaskToken(token string) string {
	if len(token) < 8 {
		return "*****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
