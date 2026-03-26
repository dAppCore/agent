// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
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
