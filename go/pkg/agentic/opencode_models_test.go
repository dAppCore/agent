// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestOpencodeParseModels_Good_FreeAndGoTiers(t *testing.T) {
	raw := "opencode/big-pickle\n" +
		"opencode/deepseek-v4-flash-free\n" +
		"opencode-go/deepseek-v4-pro\n" +
		"opencode-go/glm-5.1\n"

	models := OpencodeParseModels(raw)

	core.AssertEqual(t, 4, len(models))

	// Free OpenCode Zen tier is flagged Free; the authed Go tier is not.
	core.AssertEqual(t, "opencode", models[0].Provider)
	core.AssertEqual(t, "big-pickle", models[0].Model)
	core.AssertEqual(t, "opencode/big-pickle", models[0].ID)
	core.AssertTrue(t, models[0].Free)

	core.AssertEqual(t, "opencode-go", models[2].Provider)
	core.AssertEqual(t, "deepseek-v4-pro", models[2].Model)
	core.AssertEqual(t, "opencode-go/deepseek-v4-pro", models[2].ID)
	core.AssertFalse(t, models[2].Free)
}

func TestOpencodeParseModels_Bad_DropsOtherProviders(t *testing.T) {
	// omlx (local MLX) + huggingface are dispatchable but tracked elsewhere —
	// the OpenCode capacity surface drops them.
	raw := "omlx/Qwen3.6-27B-mxfp8\n" +
		"huggingface/deepseek-ai/DeepSeek-V4-Pro\n" +
		"opencode-go/kimi-k2.6\n"

	models := OpencodeParseModels(raw)

	core.AssertEqual(t, 1, len(models))
	core.AssertEqual(t, "opencode-go/kimi-k2.6", models[0].ID)
}

func TestOpencodeParseModels_Ugly_BlankAndMalformedLines(t *testing.T) {
	// Blank lines, a bare provider with no model, a leading-slash orphan, and a
	// trailing slash are all skipped without panicking; a whitespace-padded
	// valid id still parses.
	raw := "\n" +
		"  \n" +
		"opencode\n" + // no slash
		"opencode/\n" + // trailing slash, no model
		"/orphan\n" + // leading slash, no provider
		"  opencode-go/qwen3.7-max  \n" // padded but valid

	models := OpencodeParseModels(raw)

	core.AssertEqual(t, 1, len(models))
	core.AssertEqual(t, "opencode-go/qwen3.7-max", models[0].ID)
	core.AssertEqual(t, "qwen3.7-max", models[0].Model)
	core.AssertFalse(t, models[0].Free)
}
