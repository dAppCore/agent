// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

// OpencodeModel is one model the host's opencode exposes for dispatch — the
// full "provider/model" id a brief targets verbatim (agent: opencode:<id>).
// Only the dispatchable OpenCode tiers are surfaced: the free Zen tier
// (provider "opencode") and the authed Go tier (provider "opencode-go").
type OpencodeModel struct {
	Provider string `json:"provider"` // "opencode" (free Zen) | "opencode-go" (authed)
	Model    string `json:"model"`    // model id within the provider
	ID       string `json:"id"`       // "provider/model" — the dispatch profile
	Free     bool   `json:"free"`     // true for the free OpenCode Zen tier
}

// opencodeDispatchTiers names the opencode providers core/agent surfaces for
// capacity planning. omlx (local MLX) and huggingface are dispatchable too but
// carry their own capacity story — keep this list to the OpenCode Zen / Go
// quotas the operator tops up.
var opencodeDispatchTiers = map[string]bool{
	"opencode":    true, // free OpenCode Zen
	"opencode-go": true, // authed Go tier
}

// OpencodeParseModels turns `opencode models` output (one provider/model id per
// line) into the dispatchable OpenCode Zen (free) + Go (authed) models, in
// input order. Other providers (omlx, huggingface, …) are dropped — they carry
// their own capacity story. As the operator adds an OpenCode provider, it
// appears here with no code change: the capacity-planning surface tracks the
// live config.
//
//	models := OpencodeParseModels("opencode/big-pickle\nopencode-go/glm-5\nomlx/x")
//	// → [{opencode big-pickle opencode/big-pickle true}
//	//    {opencode-go glm-5 opencode-go/glm-5 false}]
func OpencodeParseModels(raw string) []OpencodeModel {
	var models []OpencodeModel
	for _, line := range core.Split(raw, "\n") {
		id := core.Trim(line)
		if id == "" {
			continue
		}
		slash := core.Index(id, "/")
		if slash <= 0 || slash >= len(id)-1 {
			continue // not a "provider/model" id
		}
		provider := id[:slash]
		if !opencodeDispatchTiers[provider] {
			continue
		}
		models = append(models, OpencodeModel{
			Provider: provider,
			Model:    id[slash+1:],
			ID:       id,
			Free:     provider == "opencode",
		})
	}
	return models
}

// OpencodeHostModels runs the operator's `opencode models` and returns the
// dispatchable OpenCode Zen + Go models. The enumeration is host-side — it
// reads the operator's own opencode config + auth, the same source a
// containerised `opencode run` dispatches against — so what this lists is
// exactly what `agent: opencode:<id>` can target.
//
//	models, err := OpencodeHostModels(ctx, c)
func OpencodeHostModels(ctx context.Context, c *core.Core) ([]OpencodeModel, error) {
	if c == nil {
		return nil, core.E("agentic.opencodeModels", "core unavailable", nil)
	}
	r := c.Process().Run(ctx, "opencode", "models")
	if !r.OK {
		return nil, core.E("agentic.opencodeModels", "opencode models failed", nil)
	}
	raw, _ := r.Value.(string)
	return OpencodeParseModels(raw), nil
}
