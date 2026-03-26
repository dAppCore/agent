// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestProcessregister_Register_Good(t *testing.T) {
	c := core.New(core.WithService(ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	assert.True(t, c.Process().Exists())
}

func TestProcessregister_NilCore_Bad_NilCore(t *testing.T) {
	// ProcessRegister delegates to process.Register
	// which needs a valid Core — verify it doesn't panic
	assert.NotPanics(t, func() {
		c := core.New()
		_ = ProcessRegister(c)
	})
}

func TestProcessregister_Actions_Ugly_ActionsRegistered(t *testing.T) {
	c := core.New(core.WithService(ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	assert.True(t, c.Action("process.run").Exists())
	assert.True(t, c.Action("process.start").Exists())
	assert.True(t, c.Action("process.kill").Exists())
}
