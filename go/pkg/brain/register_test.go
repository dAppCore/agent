// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	core "dappco.re/go"
)

func TestRegister_Register_Good(t *testing.T) {
	c := core.New(core.WithService(Register))
	core.AssertContains(t, c.Services(), "brain")
	core.AssertTrue(t, c.Service("brain").OK)
}

func TestRegister_Register_Bad_ServiceName(t *testing.T) {
	c := core.New(core.WithService(Register))
	core.AssertContains(t, c.Services(), "brain")
	core.AssertNotContains(t, c.Services(), "memory")
}

func TestRegister_Register_Ugly_ServiceAccessible(t *testing.T) {
	c := core.New(core.WithService(Register))
	svc := c.Service("brain")
	core.AssertTrue(t, svc.OK)
}

func TestRegister_Register_Bad(t *testing.T) {
	c := core.New(core.WithService(Register))
	core.AssertContains(t, c.Services(), "brain")
	core.AssertNotContains(t, c.Services(), "memory")
}

func TestRegister_Register_Ugly(t *testing.T) {
	c := core.New(core.WithService(Register))
	svc := c.Service("brain")
	core.AssertTrue(t, svc.OK)
}
