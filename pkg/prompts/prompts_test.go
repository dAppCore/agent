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

func TestTask_Bad_NotFound(t *testing.T) {
	_, err := Task("nonexistent")
	assert.Error(t, err)
}

func TestTemplate_Good_BackwardsCompat(t *testing.T) {
	// Template() should find prompts
	content, err := Template("coding")
	require.NoError(t, err)
	assert.Contains(t, content, "SANDBOX")

	// Template() should also find tasks
	content, err = Template("bug-fix")
	require.NoError(t, err)
	assert.Contains(t, content, "name:")
}

func TestFlow_Good(t *testing.T) {
	content, err := Flow("prod-push-polish")
	require.NoError(t, err)
	assert.True(t, len(content) > 0)
}

func TestListPrompts_Good(t *testing.T) {
	list := ListPrompts()
	assert.Contains(t, list, "coding")
	assert.Contains(t, list, "verify")
}

func TestListTasks_Good(t *testing.T) {
	list := ListTasks()
	assert.Contains(t, list, "bug-fix")
	assert.Contains(t, list, "refactor")
}

func TestPersona_Good(t *testing.T) {
	content, err := Persona("secops/developer")
	require.NoError(t, err)
	assert.Contains(t, content, "name:")
	assert.Contains(t, content, "Security")
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

func TestListTemplates_Good(t *testing.T) {
	templates := ListTemplates()
	assert.True(t, len(templates) >= 10, "expected at least 10 templates, got %d", len(templates))
	assert.Contains(t, templates, "bug-fix")
	assert.Contains(t, templates, "code-review")
}

func TestListPersonas_Good(t *testing.T) {
	personas := ListPersonas()
	assert.True(t, len(personas) >= 90, "expected at least 90 personas, got %d", len(personas))

	// Check cross-domain security roles exist
	hasSecDev := false
	hasSMMSec := false
	for _, p := range personas {
		if p == "secops/developer" {
			hasSecDev = true
		}
		if p == "smm/security-developer" {
			hasSMMSec = true
		}
	}
	assert.True(t, hasSecDev, "secops/developer not found")
	assert.True(t, hasSMMSec, "smm/security-developer not found")
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
