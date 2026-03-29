// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_HandleDispatch_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "fix tests"},
	))
	// Will fail (no local clone) but exercises the handler path
	assert.False(t, r.OK)
}

func TestActions_HandleStatus_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := newPrepWithProcess()
	r := s.handleStatus(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
}

func TestActions_HandlePrompt_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handlePrompt(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "coding"},
	))
	assert.True(t, r.OK)
}

func TestActions_HandlePrompt_Bad(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handlePrompt(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "does-not-exist"},
	))
	assert.False(t, r.OK)
}

func TestActions_HandleTask_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleTask(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "bug-fix"},
	))
	assert.True(t, r.OK)
}

func TestActions_HandleFlow_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleFlow(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "go"},
	))
	assert.True(t, r.OK)
}

func TestActions_HandlePersona_Good(t *testing.T) {
	personas := lib.ListPersonas()
	if len(personas) == 0 {
		t.Skip("no personas embedded")
	}

	s := newPrepWithProcess()
	r := s.handlePersona(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: personas[0]},
	))
	assert.True(t, r.OK)
}

func TestActions_HandlePoke_Good(t *testing.T) {
	s := newPrepWithProcess()
	s.pokeCh = make(chan struct{}, 1)
	r := s.handlePoke(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
}

func TestActions_HandleQA_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleQA(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestActions_HandleVerify_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleVerify(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestActions_HandleIngest_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleIngest(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestActions_HandleWorkspaceQuery_Good(t *testing.T) {
	s := newPrepWithProcess()
	s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	s.workspaces.Set("core/go-io/task-42", &WorkspaceStatus{Status: "blocked", Repo: "go-io"})

	r := s.handleWorkspaceQuery(nil, WorkspaceQuery{Status: "blocked"})
	require.True(t, r.OK)

	names, ok := r.Value.([]string)
	require.True(t, ok)
	require.Len(t, names, 1)
	assert.Equal(t, "core/go-io/task-42", names[0])
}

func TestActions_HandleWorkspaceQuery_Bad(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleWorkspaceQuery(nil, "not-a-workspace-query")
	assert.False(t, r.OK)
	assert.Nil(t, r.Value)
}
