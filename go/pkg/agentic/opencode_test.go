// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestOpenCode_Profile_Good_GemmaAgentic(t *testing.T) {
	profile := opencodeProfileConfig("gemma4-agentic")

	core.AssertEqual(t, "core-local", profile.Provider)
	core.AssertEqual(t, "http://127.0.0.1:8001/v1", profile.BaseURL)
	core.AssertEqual(t, "google/gemma-4-26B-A4B-it", profile.Model)
}

func TestOpenCode_Profile_Good_GemmaLlamaCpp(t *testing.T) {
	profile := opencodeProfileConfig("gemma4-llamacpp")

	core.AssertEqual(t, "http://127.0.0.1:8080/v1", profile.BaseURL)
	core.AssertEqual(t, "gemma-4-26B-A4B-it-UD-Q8_K_XL.gguf", profile.Model)
	core.AssertEqual(t, "gemma-4-26B-A4B-it-UD-Q8_K_XL.gguf", profile.SmallModel)
}

func TestOpenCode_Profile_Good_LemerChatter(t *testing.T) {
	profile := opencodeProfileConfig("lemer-chatter")

	core.AssertEqual(t, "core-mlx", profile.Provider)
	core.AssertEqual(t, "http://127.0.0.1:8007/v1", profile.BaseURL)
	core.AssertEqual(t, "lthn/lemer-mlx-bf16", profile.Model)
	core.AssertEqual(t, "lthn/lemer-mlx-bf16", profile.SmallModel)
}

func TestOpenCode_Profile_Good_GemmaMLXAgentic(t *testing.T) {
	profile := opencodeProfileConfig("gemma4-mlx-agentic")

	core.AssertEqual(t, "core-mlx", profile.Provider)
	core.AssertEqual(t, "http://127.0.0.1:8001/v1", profile.BaseURL)
	core.AssertEqual(t, "mlx-community/gemma-4-26b-a4b-it-4bit", profile.Model)
	core.AssertEqual(t, "lthn/lemer-mlx-bf16", profile.SmallModel)
}

func TestOpenCode_Profile_Good_GemmaVLLMMTP(t *testing.T) {
	profile := opencodeProfileConfig("gemma4-vllm-mtp")

	core.AssertEqual(t, "core-vllm", profile.Provider)
	core.AssertEqual(t, "http://127.0.0.1:8008/v1", profile.BaseURL)
	core.AssertEqual(t, "google/gemma-4-26B-A4B-it", profile.Model)
	core.AssertEqual(t, "google/gemma-4-26B-A4B-it", profile.SmallModel)
}

func TestOpenCode_Profile_Good_EnvOverrides(t *testing.T) {
	t.Setenv("CORE_OPENCODE_GEMMA4_AGENTIC_BASE_URL", "http://127.0.0.1:9001/v1")
	t.Setenv("CORE_OPENCODE_GEMMA4_AGENTIC_MODEL", "lthn/lemma-gemma-4-26b")

	profile := opencodeProfileConfig("gemma4-agentic")

	core.AssertEqual(t, "http://127.0.0.1:9001/v1", profile.BaseURL)
	core.AssertEqual(t, "lthn/lemma-gemma-4-26b", profile.Model)
}

func TestOpenCode_Profile_Good_LemmaFineTune(t *testing.T) {
	profile := opencodeProfileConfig("lemma")

	core.AssertEqual(t, "http://127.0.0.1:8006/v1", profile.BaseURL)
	core.AssertEqual(t, "lthn/lemma", profile.Model)
}

func TestOpenCode_Profile_Good_GemmaSmallModels(t *testing.T) {
	chatter := opencodeProfileConfig("gemma4-chatter")
	e4b := opencodeProfileConfig("gemma4-e4b")

	core.AssertEqual(t, "google/gemma-4-E2B-it", chatter.Model)
	core.AssertEqual(t, "google/gemma-4-E4B-it", e4b.Model)
}

func TestOpenCode_Command_Good_GemmaAgentic(t *testing.T) {
	script := opencodeAgentCommandScript("gemma4-agentic", "fix tests")

	core.AssertContains(t, script, "OPENCODE_CONFIG_CONTENT=")
	core.AssertContains(t, script, "opencode run")
	core.AssertContains(t, script, "--dangerously-skip-permissions")
	core.AssertContains(t, script, "--model")
	core.AssertContains(t, script, "core-local/google/gemma-4-26B-A4B-it")
	core.AssertContains(t, script, "'fix tests'")
}

func TestOpenCode_Command_Good_LemerChatter(t *testing.T) {
	script := opencodeAgentCommandScript("lemer", "chat")

	core.AssertContains(t, script, "core-mlx/lthn/lemer-mlx-bf16")
	core.AssertContains(t, script, "http://127.0.0.1:8007/v1")
}

func TestOpenCode_Command_Ugly_ShellQuoting(t *testing.T) {
	script := opencodeAgentCommandScript("gemma4-agentic", "can't break")

	core.AssertContains(t, script, "'can'\\''t break'")
}
