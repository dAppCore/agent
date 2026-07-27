// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func newLocalModeSubsystem(t *testing.T) (*PrepSubsystem, *core.Core) {
	t.Helper()
	c := core.New()
	c.Config().Enable("auto-pr")
	c.Config().Enable("auto-merge")
	c.Config().Enable("auto-ingest")
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	return s, c
}

func TestPrepSubsystem_ApplyDispatchLocalMode_Good_DisablesOutwardActions(t *testing.T) {
	s, c := newLocalModeSubsystem(t)

	applied := s.applyDispatchLocalMode(core.NewOptions(core.Option{Key: "no-pr", Value: true}))

	core.AssertTrue(t, applied)
	core.AssertFalse(t, c.Config().Enabled("auto-pr"))
	core.AssertFalse(t, c.Config().Enabled("auto-merge"))
	core.AssertFalse(t, c.Config().Enabled("auto-ingest"))
}

func TestPrepSubsystem_ApplyDispatchLocalMode_Bad_NoFlagLeavesConfig(t *testing.T) {
	s, c := newLocalModeSubsystem(t)

	applied := s.applyDispatchLocalMode(core.NewOptions())

	core.AssertFalse(t, applied)
	// Without --no-pr the outward actions stay as configured (auto-pr on).
	core.AssertTrue(t, c.Config().Enabled("auto-pr"))
	core.AssertTrue(t, c.Config().Enabled("auto-merge"))
}

func TestPrepSubsystem_ApplyDispatchLocalMode_Ugly_NilRuntimeNoPanic(t *testing.T) {
	// A subsystem with no ServiceRuntime (and a nil receiver) must not panic
	// trying to reach Config() — it simply reports local mode not applied.
	var nilSubsystem *PrepSubsystem
	core.AssertFalse(t, nilSubsystem.applyDispatchLocalMode(core.NewOptions(core.Option{Key: "no-pr", Value: true})))

	bare := &PrepSubsystem{}
	core.AssertFalse(t, bare.applyDispatchLocalMode(core.NewOptions(core.Option{Key: "no-pr", Value: true})))
}
