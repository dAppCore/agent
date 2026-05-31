// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

// opencodeModels lists the OpenCode models the host can dispatch against — the
// free Zen tier and the authed Go tier — read live from the operator's
// `opencode models`. This is the capacity-planning surface: every id printed
// can be targeted as `agent: opencode:<id>`, and a provider added to the
// operator's opencode config shows up here with no code change.
//
//	core-agent opencode-models
func (commands applicationCommandSet) opencodeModels(_ core.Options) core.Result {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	models, err := agentic.OpencodeHostModels(ctx, commands.coreApp)
	if err != nil {
		applicationPrint("opencode-models: %v", err)
		return core.Result{}
	}
	if len(models) == 0 {
		applicationPrint("opencode-models: none — is opencode installed and authed? (opencode auth login)")
		return core.Result{}
	}

	var free, paid []agentic.OpencodeModel
	for _, model := range models {
		if model.Free {
			free = append(free, model)
			continue
		}
		paid = append(paid, model)
	}

	applicationPrint("OpenCode dispatch models — target as `agent: opencode:<id>`")
	applicationPrint("")
	applicationPrint("free (OpenCode Zen) — %d:", len(free))
	for _, model := range free {
		applicationPrint("  opencode:%s", model.ID)
	}
	applicationPrint("")
	applicationPrint("go (authed) — %d:", len(paid))
	for _, model := range paid {
		applicationPrint("  opencode:%s", model.ID)
	}
	return core.Result{OK: true}
}
