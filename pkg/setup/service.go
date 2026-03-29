// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"

	core "dappco.re/go/core"
)

// SetupOptions carries service-level setup configuration.
//
//	opts := setup.SetupOptions{}
type SetupOptions struct{}

// Service exposes workspace setup through Core service registration.
//
//	c := core.New(core.WithService(setup.Register))
//	svc, _ := core.ServiceFor[*setup.Service](c, "setup")
type Service struct {
	*core.ServiceRuntime[SetupOptions]
}

// Register wires the setup service into Core.
//
//	c := core.New(core.WithService(setup.Register))
//	svc, _ := core.ServiceFor[*setup.Service](c, "setup")
func Register(c *core.Core) core.Result {
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{}),
	}
	return core.Result{Value: svc, OK: true}
}

// OnStartup keeps the setup service ready for Core startup hooks.
//
//	result := svc.OnStartup(context.Background())
func (s *Service) OnStartup(ctx context.Context) core.Result {
	return core.Result{OK: true}
}

// DetectGitRemote reads `origin` and returns `owner/repo` when available.
//
//	remote := svc.DetectGitRemote("/srv/repos/agent")
func (s *Service) DetectGitRemote(path string) string {
	r := s.Core().Process().RunIn(context.Background(), path, "git", "remote", "get-url", "origin")
	if !r.OK {
		return ""
	}
	return parseGitRemote(core.Trim(r.Value.(string)))
}
