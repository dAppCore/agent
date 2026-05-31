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

func TestOpenCode_Profile_Good_GemmaMLXMTP(t *testing.T) {
	profile := opencodeProfileConfig("gemma4-mlx-mtp")

	core.AssertEqual(t, "core-mlx", profile.Provider)
	core.AssertEqual(t, "http://127.0.0.1:8010/v1", profile.BaseURL)
	core.AssertEqual(t, "mlx-community/gemma-4-26b-a4b-it-4bit", profile.Model)
	core.AssertEqual(t, "mlx-community/gemma-4-26b-a4b-it-4bit", profile.SmallModel)
}

func TestOpenCode_Profile_Good_GemmaMLXXHighMTP(t *testing.T) {
	profile := opencodeProfileConfig("gemma4-mlx-xhigh-mtp")

	core.AssertEqual(t, "core-mlx", profile.Provider)
	core.AssertEqual(t, "http://127.0.0.1:8011/v1", profile.BaseURL)
	core.AssertEqual(t, "mlx-community/gemma-4-31b-it-4bit", profile.Model)
	core.AssertEqual(t, "mlx-community/gemma-4-31b-it-4bit", profile.SmallModel)
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

func TestOpenCode_Command_Good_HostModelTakesHostDefaults(t *testing.T) {
	script := opencodeAgentCommandScript("opencode/deepseek-v4-flash-free", "fix tests")

	// Host-config model: no core-local provider block — opencode uses the
	// operator's own auth/config and the model id passes through verbatim.
	if core.Contains(script, "OPENCODE_CONFIG_CONTENT=") {
		t.Errorf("host model must not inject a core-local provider config; got: %s", script)
	}
	core.AssertContains(t, script, "opencode run")
	core.AssertContains(t, script, "--dangerously-skip-permissions")
	core.AssertContains(t, script, "--model 'opencode/deepseek-v4-flash-free'")
	core.AssertContains(t, script, "'fix tests'")
	// opencode runs host-native, so no credential prelude/scratch path is
	// emitted — opencode reads the operator's own auth.json in place.
	core.AssertNotContains(t, script, "/run/oc-auth.json")
}

func TestOpenCode_Command_Good_HostModelGoTier(t *testing.T) {
	script := opencodeAgentCommandScript("opencode-go/deepseek-v4-pro", "review")

	if core.Contains(script, "OPENCODE_CONFIG_CONTENT=") {
		t.Errorf("Go-tier host model must not inject config; got: %s", script)
	}
	core.AssertContains(t, script, "--model 'opencode-go/deepseek-v4-pro'")
}

func TestOpenCode_IsHostModel(t *testing.T) {
	core.AssertEqual(t, true, opencodeIsHostModel("opencode/deepseek-v4-flash-free"))
	core.AssertEqual(t, true, opencodeIsHostModel("omlx/Qwen3.6-27B-mxfp8"))
	core.AssertEqual(t, false, opencodeIsHostModel("gemma4-agentic"))
	core.AssertEqual(t, false, opencodeIsHostModel(""))
}
