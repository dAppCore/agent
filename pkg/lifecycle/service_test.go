package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultServiceOptions_Good(t *testing.T) {
	opts := DefaultServiceOptions()

	assert.Equal(t, []string{"Bash", "Read", "Glob", "Grep"}, opts.DefaultTools)
	assert.False(t, opts.AllowEdit, "default should not allow edit")
}

func TestTaskPrompt_SetGetTaskID(t *testing.T) {
	tp := &TaskPrompt{
		Prompt:  "test prompt",
		WorkDir: "/tmp",
	}

	assert.Empty(t, tp.GetTaskID(), "should start empty")

	tp.SetTaskID("task-abc-123")
	assert.Equal(t, "task-abc-123", tp.GetTaskID())

	tp.SetTaskID("task-def-456")
	assert.Equal(t, "task-def-456", tp.GetTaskID(), "should allow overwriting")
}

func TestTaskPrompt_SetGetTaskID_Empty(t *testing.T) {
	tp := &TaskPrompt{}

	tp.SetTaskID("")
	assert.Empty(t, tp.GetTaskID())
}

func TestTaskCommit_Fields(t *testing.T) {
	tc := TaskCommit{
		Path:    "/home/user/project",
		Name:    "test-commit",
		CanEdit: true,
	}

	assert.Equal(t, "/home/user/project", tc.Path)
	assert.Equal(t, "test-commit", tc.Name)
	assert.True(t, tc.CanEdit)
}

func TestTaskCommit_DefaultCanEdit(t *testing.T) {
	tc := TaskCommit{
		Path: "/tmp",
		Name: "no-edit",
	}

	assert.False(t, tc.CanEdit, "default should be false")
}

func TestServiceOptions_CustomTools(t *testing.T) {
	opts := ServiceOptions{
		DefaultTools: []string{"Bash", "Read", "Write", "Edit"},
		AllowEdit:    true,
	}

	assert.Len(t, opts.DefaultTools, 4)
	assert.True(t, opts.AllowEdit)
}

func TestTaskPrompt_AllFields(t *testing.T) {
	tp := TaskPrompt{
		Prompt:       "Refactor the authentication module",
		WorkDir:      "/home/user/project",
		AllowedTools: []string{"Bash", "Read", "Edit"},
	}

	assert.Equal(t, "Refactor the authentication module", tp.Prompt)
	assert.Equal(t, "/home/user/project", tp.WorkDir)
	assert.Equal(t, []string{"Bash", "Read", "Edit"}, tp.AllowedTools)
}
