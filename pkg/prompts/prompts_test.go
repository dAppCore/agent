// SPDX-License-Identifier: EUPL-1.2

package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrompt_Good(t *testing.T) {
	content, err := Prompt("coding")
	require.NoError(t, err)
	assert.Contains(t, content, "SANDBOX")
	assert.Contains(t, content, "Closeout Sequence")
}

func TestPrompt_Bad_NotFound(t *testing.T) {
	_, err := Prompt("nonexistent")
	assert.Error(t, err)
}

func TestTask_Good(t *testing.T) {
	content, err := Task("bug-fix")
	require.NoError(t, err)
	assert.Contains(t, content, "name:")
}

func TestTask_Good_Nested(t *testing.T) {
	content, err := Task("code/review")
	require.NoError(t, err)
	assert.Contains(t, content, "Code Review")
}

func TestTaskBundle_Good(t *testing.T) {
	main, bundle, err := TaskBundle("code/review")
	require.NoError(t, err)
	assert.Contains(t, main, "Code Review")
	assert.NotNil(t, bundle)
	assert.Contains(t, bundle, "conventions.md")
	assert.Contains(t, bundle, "severity.md")
	assert.Contains(t, bundle["conventions.md"], "coreerr.E")
}

func TestTaskBundle_Good_NoBundleDir(t *testing.T) {
	main, bundle, err := TaskBundle("bug-fix")
	require.NoError(t, err)
	assert.Contains(t, main, "name:")
	assert.Nil(t, bundle)
}

func TestTask_Bad_NotFound(t *testing.T) {
	_, err := Task("nonexistent")
	assert.Error(t, err)
}

func TestTemplate_Good_BackwardsCompat(t *testing.T) {
	content, err := Template("coding")
	require.NoError(t, err)
	assert.Contains(t, content, "SANDBOX")

	content, err = Template("bug-fix")
	require.NoError(t, err)
	assert.Contains(t, content, "name:")
}

func TestFlow_Good(t *testing.T) {
	content, err := Flow("go")
	require.NoError(t, err)
	assert.Contains(t, content, "go build")
}

func TestFlow_Good_Docker(t *testing.T) {
	content, err := Flow("docker")
	require.NoError(t, err)
	assert.Contains(t, content, "docker build")
}

func TestPersona_Good(t *testing.T) {
	content, err := Persona("secops/developer")
	require.NoError(t, err)
	assert.Contains(t, content, "name:")
}

func TestPersona_Good_SMM(t *testing.T) {
	content, err := Persona("smm/security-developer")
	require.NoError(t, err)
	assert.Contains(t, content, "OAuth")
}

func TestPersona_Bad_NotFound(t *testing.T) {
	_, err := Persona("nonexistent/persona")
	assert.Error(t, err)
}

func TestListPrompts_Good(t *testing.T) {
	list := ListPrompts()
	assert.Contains(t, list, "coding")
	assert.Contains(t, list, "verify")
	assert.True(t, len(list) >= 5)
}

func TestListTasks_Good(t *testing.T) {
	list := ListTasks()
	assert.Contains(t, list, "bug-fix")
	// Nested tasks
	hasCodeReview := false
	for _, t := range list {
		if t == "code/review" {
			hasCodeReview = true
		}
	}
	assert.True(t, hasCodeReview, "code/review not found in tasks")
}

func TestListFlows_Good(t *testing.T) {
	list := ListFlows()
	assert.Contains(t, list, "go")
	assert.Contains(t, list, "php")
	assert.Contains(t, list, "docker")
	assert.True(t, len(list) >= 9)
}

func TestListPersonas_Good(t *testing.T) {
	personas := ListPersonas()
	assert.True(t, len(personas) >= 90)
}

func TestListPersonas_Good_NoPrefixDuplication(t *testing.T) {
	for _, p := range ListPersonas() {
		parts := strings.Split(p, "/")
		if len(parts) == 2 {
			domain := parts[0]
			file := parts[1]
			assert.False(t, strings.HasPrefix(file, domain+"-"),
				"persona %q has redundant domain prefix in filename", p)
		}
	}
}
