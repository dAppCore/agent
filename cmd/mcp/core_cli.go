package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

var allowedCorePrefixes = map[string]struct{}{
	"dev":   {},
	"go":    {},
	"php":   {},
	"build": {},
}

func coreCliHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, err := request.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError("command is required"), nil
	}

	args := request.GetStringSlice("args", nil)
	base, mergedArgs, err := normalizeCoreCommand(command, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result := runCoreCommand(execCtx, base, mergedArgs)
	return mcp.NewToolResultStructuredOnly(result), nil
}

func normalizeCoreCommand(command string, args []string) (string, []string, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil, errors.New("command cannot be empty")
	}

	base := parts[0]
	if _, ok := allowedCorePrefixes[base]; !ok {
		return "", nil, fmt.Errorf("command not allowed: %s", base)
	}

	merged := append([]string{}, parts[1:]...)
	merged = append(merged, args...)

	return base, merged, nil
}

func runCoreCommand(ctx context.Context, command string, args []string) CoreCliResult {
	cmd := exec.CommandContext(ctx, "core", append([]string{command}, args...)...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) {
			exitCode = 124
		}
		if errors.Is(err, context.DeadlineExceeded) {
			if stderr.Len() > 0 {
				stderr.WriteString("\n")
			}
			stderr.WriteString("command timed out after 30s")
		}
	}

	return CoreCliResult{
		Command:  command,
		Args:     args,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}
