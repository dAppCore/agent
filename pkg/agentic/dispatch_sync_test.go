// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDispatchsync_ContainerCommand_Good(t *testing.T) {
	cmd, args := containerCommand("codex", []string{"--model", "gpt-5.4"}, "/workspace/task-5", "/workspace/task-5/.meta")
	assert.Equal(t, "docker", cmd)
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "/workspace/task-5:/workspace")
	assert.Contains(t, args, "/workspace/task-5/.meta:/workspace/.meta")
	assert.Contains(t, args, "/workspace/repo")
}

func TestDispatchsync_ContainerCommand_Bad_UnknownAgent(t *testing.T) {
	cmd, args := containerCommand("unknown", nil, "/workspace/task-5", "/workspace/task-5/.meta")
	assert.Equal(t, "docker", cmd)
	assert.NotEmpty(t, args)
}

func TestDispatchsync_ContainerCommand_Ugly_EmptyArgs(t *testing.T) {
	assert.NotPanics(t, func() {
		containerCommand("codex", nil, "", "")
	})
}
