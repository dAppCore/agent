// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestRegister_Register_Good(t *testing.T) {
	c := core.New(core.WithService(Register))
	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	assert.True(t, ok)
	assert.NotNil(t, svc)
}

func TestRegister_Register_Bad_ServiceName(t *testing.T) {
	c := core.New(core.WithService(Register))
	assert.Contains(t, c.Services(), "monitor")
}

func TestRegister_Register_Ugly_ServiceRuntime(t *testing.T) {
	c := core.New(core.WithService(Register))
	svc, _ := core.ServiceFor[*Subsystem](c, "monitor")
	assert.NotNil(t, svc.ServiceRuntime)
	assert.Equal(t, c, svc.Core())
}
