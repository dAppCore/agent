// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"

	core "dappco.re/go/core"
)

// RuntimeOptions carries service-level setup configuration.
//
//	options := setup.RuntimeOptions{}
type RuntimeOptions struct{}

// Service exposes workspace setup through Core service registration.
//
//	c := core.New(core.WithService(setup.Register))
//	service, _ := core.ServiceFor[*setup.Service](c, "setup")
type Service struct {
	*core.ServiceRuntime[RuntimeOptions]
}

// Register wires the setup service into Core.
//
//	c := core.New(core.WithService(setup.Register))
//	service, _ := core.ServiceFor[*setup.Service](c, "setup")
func Register(c *core.Core) core.Result {
	service := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{}),
	}
	return core.Result{Value: service, OK: true}
}

// OnStartup keeps the setup service ready for Core startup hooks.
//
//	result := service.OnStartup(context.Background())
func (s *Service) OnStartup(ctx context.Context) core.Result {
	return core.Result{OK: true}
}

// DetectGitRemote reads `origin` and returns `owner/repo` when available.
//
//	remote := service.DetectGitRemote("/srv/repos/agent")
func (s *Service) DetectGitRemote(path string) string {
	result := s.Core().Process().RunIn(context.Background(), path, "git", "remote", "get-url", "origin")
	if !result.OK {
		return ""
	}
	return parseGitRemote(core.Trim(result.Value.(string)))
}
