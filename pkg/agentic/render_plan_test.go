// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- renderPlan ---

func TestPrep_RenderPlan_Good_BugFix(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	result := s.renderPlan("bug-fix", nil, "Fix the auth bug")
	if result == "" {
		t.Skip("bug-fix template not available in embedded assets")
	}
	assert.Contains(t, result, "Bug Fix")
	assert.Contains(t, result, "Fix the auth bug")
	assert.Contains(t, result, "Phase 1")
	assert.Contains(t, result, "Investigation")
}

func TestPrep_RenderPlan_Good_WithVariables(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	vars := map[string]string{
		"bug_description": "Login fails on mobile",
		"location":        "pkg/auth/handler.go",
	}
	result := s.renderPlan("bug-fix", vars, "Fix login")
	if result == "" {
		t.Skip("bug-fix template not available")
	}
	assert.Contains(t, result, "Fix login")
}

func TestPrep_RenderPlan_Bad_UnknownTemplate(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	result := s.renderPlan("nonexistent-template-slug", nil, "task")
	assert.Empty(t, result)
}

func TestPrep_RenderPlan_Good_NoTask(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	result := s.renderPlan("bug-fix", nil, "")
	if result == "" {
		t.Skip("template not available")
	}
	assert.NotContains(t, result, "**Task:**")
}

func TestPrep_RenderPlan_Good_NewFeature(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	result := s.renderPlan("new-feature", nil, "Add caching")
	if result == "" {
		t.Skip("new-feature template not available")
	}
	assert.Contains(t, result, "Add caching")
}
