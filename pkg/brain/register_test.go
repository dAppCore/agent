// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestRegister_Register_Good(t *testing.T) {
	c := core.New(core.WithService(Register))
	assert.Contains(t, c.Services(), "brain")
}

func TestRegister_Register_Bad_ServiceName(t *testing.T) {
	c := core.New(core.WithService(Register))
	assert.Contains(t, c.Services(), "brain")
}

func TestRegister_Register_Ugly_ServiceAccessible(t *testing.T) {
	c := core.New(core.WithService(Register))
	svc := c.Service("brain")
	assert.True(t, svc.OK)
}
