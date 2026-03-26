// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"

	core "dappco.re/go/core"
)

// SetupOptions configures the setup service.
type SetupOptions struct{}

// Service provides workspace setup and scaffolding as a Core service.
// Registers as "setup" — use s.Core().Process() for git operations.
//
//	core.New(core.WithService(setup.Register))
type Service struct {
	*core.ServiceRuntime[SetupOptions]
}

// Register is the WithService factory for setup.
//
//	core.New(core.WithService(setup.Register))
func Register(c *core.Core) core.Result {
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{}),
	}
	return core.Result{Value: svc, OK: true}
}

// OnStartup implements core.Startable.
func (s *Service) OnStartup(ctx context.Context) core.Result {
	return core.Result{OK: true}
}

// DetectGitRemote extracts owner/repo from git remote origin via Core Process.
//
//	remote := svc.DetectGitRemote("/repo")
func (s *Service) DetectGitRemote(path string) string {
	r := s.Core().Process().RunIn(context.Background(), path, "git", "remote", "get-url", "origin")
	if !r.OK {
		return ""
	}
	return parseGitRemote(core.Trim(r.Value.(string)))
}
