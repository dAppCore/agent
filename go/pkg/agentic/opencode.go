// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/opencode"
)

// opencodeServiceName is the Core registration name pkg/opencode binds
// under (see opencode.Service docs). The provider resolves the local
// opencode Service through this name at generate time — core/agent OWNS
// opencode, so generation is a direct in-process call, no HTTP hop.
const opencodeServiceName = "opencode"

// opencodeProviderName is the ProviderManager key for the opencode
// backend.
const opencodeProviderName = "opencode"

// opencodeDefaultModel is the DefaultModel the opencode provider reports
// when the caller does not pin one. Empty profile + empty model let
// opencode-serve fall back to the profile's configured default.
const opencodeDefaultModel = "gemma4-agentic"

// newOpencodeGenerate returns a ProviderGenerateFunc that drives
// generation through the local opencode Service. The Service is resolved
// from Core lazily on each call so the provider can be registered before
// the opencode Service finishes wiring (and degrades to a clear error
// when opencode isn't registered in this binary).
//
//	generate := newOpencodeGenerate(s.Core())
//	text, err := generate(ctx, "Draft a release note", map[string]any{"profile": "lemma"})
func newOpencodeGenerate(c *core.Core) ProviderGenerateFunc {
	return func(ctx context.Context, prompt string, options map[string]any) (string, error) {
		if c == nil {
			return "", core.E("opencode.generate", "core unavailable", nil)
		}
		svc, ok := core.ServiceFor[*opencode.Service](c, opencodeServiceName)
		if !ok || svc == nil {
			return "", core.E("opencode.generate", "opencode service not registered", nil)
		}

		input := opencode.GenerateInput{
			Prompt:    prompt,
			Profile:   optionMapString(options, "profile"),
			Model:     opencodeMessageModel(options),
			Agent:     optionMapString(options, "agent"),
			SandboxID: optionMapString(options, "sandbox_id", "sandbox-id"),
		}

		r := svc.Generate(input)
		if !r.OK {
			return "", core.E("opencode.generate", r.Error(), nil)
		}
		text, _ := r.Value.(string)
		return text, nil
	}
}

// opencodeMessageModel resolves the message model id sent to
// opencode-serve. The ProviderManager wrapper injects "model" =
// opencodeDefaultModel ("gemma4-agentic") as a sentinel when the caller
// pins nothing; that sentinel names a PROFILE, not an upstream model id,
// so it is dropped here (the profile already determines the model). A
// caller-supplied provider/model form (e.g. "core-local/lthn/lemma") is
// passed through unchanged.
func opencodeMessageModel(options map[string]any) string {
	model := optionMapString(options, "model")
	if model == "" || model == opencodeDefaultModel {
		return ""
	}
	return model
}

// optionMapString reads the first non-empty string value for any of the
// given keys out of an options map.
//
//	profile := optionMapString(options, "profile")
func optionMapString(options map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := options[key]; ok {
			if str, ok := value.(string); ok && core.Trim(str) != "" {
				return str
			}
		}
	}
	return ""
}

// newOpencodeProviderManager builds the real ProviderManager backed by
// the local opencode Service. The opencode provider is registered
// alongside the named claude/gemini/openai providers; all four route
// through the same opencode backend (opencode-serve fronts whichever
// upstream the selected profile configures), so generation is real for
// every registered name rather than the nil-generate fallback.
//
//	manager := newOpencodeProviderManager(s.Core())
//	provider, _ := manager.Provider("opencode")
//	text, _ := provider.Generate(ctx, "Draft a release note", nil)
func newOpencodeProviderManager(c *core.Core) *ProviderManager {
	generate := newOpencodeGenerate(c)
	manager := NewProviderManager(generate)
	manager.Register(newContentProvider(opencodeProviderName, opencodeDefaultModel, true, generate))
	return manager
}

type opencodeProfile struct {
	Provider   string
	BaseURL    string
	Model      string
	SmallModel string
	Agent      string
}

func opencodeProfileConfig(profile string) opencodeProfile {
	normalisedProfile := core.Lower(core.Trim(profile))
	config := opencodeProfile{
		Provider:   "core-local",
		BaseURL:    "http://127.0.0.1:8000/v1",
		Model:      normalisedProfile,
		SmallModel: "",
		Agent:      "",
	}

	switch normalisedProfile {
	case "", "gemma4-agentic":
		config.BaseURL = "http://127.0.0.1:8001/v1"
		config.Model = "google/gemma-4-26B-A4B-it"
		config.SmallModel = "google/gemma-4-E4B-it"
	case "gemma4-llamacpp", "gemma4-llama":
		config.BaseURL = "http://127.0.0.1:8080/v1"
		config.Model = "gemma-4-26B-A4B-it-UD-Q8_K_XL.gguf"
		config.SmallModel = "gemma-4-26B-A4B-it-UD-Q8_K_XL.gguf"
	case "lemer", "lemer-chatter", "chatter":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8007/v1"
		config.Model = "lthn/lemer-mlx-bf16"
		config.SmallModel = "lthn/lemer-mlx-bf16"
	case "gemma4-mlx-agentic", "gemma4-mlx-26b":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8001/v1"
		config.Model = "mlx-community/gemma-4-26b-a4b-it-4bit"
		config.SmallModel = "lthn/lemer-mlx-bf16"
	case "gemma4-mlx-mtp", "gemma4-mlx-agentic-mtp", "gemma4-mlx-26b-mtp":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8010/v1"
		config.Model = "mlx-community/gemma-4-26b-a4b-it-4bit"
		config.SmallModel = "mlx-community/gemma-4-26b-a4b-it-4bit"
	case "gemma4-mlx-xhigh", "gemma4-mlx-31b":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8002/v1"
		config.Model = "mlx-community/gemma-4-31b-it-4bit"
		config.SmallModel = "lthn/lemer-mlx-bf16"
	case "gemma4-mlx-xhigh-mtp", "gemma4-mlx-31b-mtp":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8011/v1"
		config.Model = "mlx-community/gemma-4-31b-it-4bit"
		config.SmallModel = "mlx-community/gemma-4-31b-it-4bit"
	case "gemma4-mlx-e2b":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8004/v1"
		config.Model = "mlx-community/gemma-4-e2b-it-4bit"
		config.SmallModel = "lthn/lemer-mlx-bf16"
	case "gemma4-mlx-e4b":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8005/v1"
		config.Model = "mlx-community/gemma-4-e4b-it-mxfp8"
		config.SmallModel = "lthn/lemer-mlx-bf16"
	case "gemma4-vllm-mtp", "gemma4-vllm-agentic-mtp", "gemma4-rocm-mtp":
		config.Provider = "core-vllm"
		config.BaseURL = "http://127.0.0.1:8008/v1"
		config.Model = "google/gemma-4-26B-A4B-it"
		config.SmallModel = "google/gemma-4-26B-A4B-it"
	case "gemma4-vllm-xhigh-mtp", "gemma4-rocm-xhigh-mtp":
		config.Provider = "core-vllm"
		config.BaseURL = "http://127.0.0.1:8009/v1"
		config.Model = "google/gemma-4-31B-it"
		config.SmallModel = "google/gemma-4-31B-it"
	case "gemma4-xhigh":
		config.BaseURL = "http://127.0.0.1:8002/v1"
		config.Model = "google/gemma-4-31B-it"
		config.SmallModel = "google/gemma-4-E4B-it"
	case "gemma4-chatter", "gemma4-e2b":
		config.BaseURL = "http://127.0.0.1:8004/v1"
		config.Model = "google/gemma-4-E2B-it"
		config.SmallModel = "google/gemma-4-E2B-it"
	case "gemma4-e4b":
		config.BaseURL = "http://127.0.0.1:8005/v1"
		config.Model = "google/gemma-4-E4B-it"
		config.SmallModel = "google/gemma-4-E2B-it"
	case "lemma":
		config.BaseURL = "http://127.0.0.1:8006/v1"
		config.Model = "lthn/lemma"
		config.SmallModel = "google/gemma-4-E2B-it"
	case "qwen36":
		config.BaseURL = "http://127.0.0.1:8003/v1"
		config.Model = "Qwen/Qwen3.6-35B-A3B-FP8"
		config.SmallModel = "google/gemma-4-E4B-it"
	case "qwen36-mlx":
		config.Provider = "core-mlx"
		config.BaseURL = "http://127.0.0.1:8003/v1"
		config.Model = "mlx-community/Qwen3.6-35B-A3B-4bit"
		config.SmallModel = "lthn/lemer-mlx-bf16"
	}

	envPrefix := core.Concat("CORE_OPENCODE_", opencodeProfileEnvName(normalisedProfile), "_")
	if value := core.Env(core.Concat(envPrefix, "PROVIDER")); value != "" {
		config.Provider = value
	}
	if value := core.Env(core.Concat(envPrefix, "BASE_URL")); value != "" {
		config.BaseURL = value
	}
	if value := core.Env(core.Concat(envPrefix, "MODEL")); value != "" {
		config.Model = value
	}
	if value := core.Env(core.Concat(envPrefix, "SMALL_MODEL")); value != "" {
		config.SmallModel = value
	}
	if value := core.Env(core.Concat(envPrefix, "AGENT")); value != "" {
		config.Agent = value
	}

	return config
}

func opencodeAgentCommandScript(profile, prompt string) string {
	builder := core.NewBuilder()

	// Host-defaults: a provider-prefixed profile (e.g.
	// "opencode/deepseek-v4-flash-free", "opencode-go/deepseek-v4-pro",
	// "omlx/Qwen3.6-27B-mxfp8") names a model served by the operator's own
	// opencode config + auth. Don't inject a core-local provider block — let
	// opencode read the operator's own ~/.config/opencode + auth and pass the
	// model id through verbatim. opencode dispatches run host-native (see
	// isNativeAgent), so this reads the operator's real config in place — no
	// credential copy or mount. This is the "take from host defaults" path: the
	// free OpenCode Zen / authed Go / HF / local-MLX models all flow through here.
	if opencodeIsHostModel(profile) {
		builder.WriteString("opencode run --dangerously-skip-permissions --model ")
		builder.WriteString(shellQuote(profile))
		builder.WriteString(" ")
		builder.WriteString(shellQuote(prompt))
		return builder.String()
	}

	// Core-local profile (gemma4-agentic, lemma, …): inject the narrowed
	// provider block pointing at the local inference endpoint.
	config := opencodeProfileConfig(profile)
	model := core.Concat(config.Provider, "/", config.Model)
	builder.WriteString("OPENCODE_CONFIG_CONTENT=")
	builder.WriteString(shellQuote(opencodeConfigContent(config)))
	builder.WriteString(" opencode run --dangerously-skip-permissions --model ")
	builder.WriteString(shellQuote(model))
	if config.Agent != "" {
		builder.WriteString(" --agent ")
		builder.WriteString(shellQuote(config.Agent))
	}
	builder.WriteString(" ")
	builder.WriteString(shellQuote(prompt))
	return builder.String()
}

// opencodeIsHostModel reports whether a profile is an operator-config model id
// (provider-prefixed, e.g. "opencode/deepseek-v4-flash-free") rather than a
// bare core-local profile name (e.g. "gemma4-agentic"). Host models route
// through the operator's own opencode auth/config; core-local profiles get a
// generated provider block.
//
//	opencodeIsHostModel("opencode/deepseek-v4-flash-free")  // true
//	opencodeIsHostModel("gemma4-agentic")                   // false
func opencodeIsHostModel(profile string) bool {
	return core.Contains(profile, "/")
}

func opencodeConfigContent(config opencodeProfile) string {
	models := map[string]any{
		config.Model: map[string]any{
			"name": config.Model,
		},
	}
	if config.SmallModel != "" {
		models[config.SmallModel] = map[string]any{
			"name": config.SmallModel,
		}
	}

	content := map[string]any{
		"$schema":    "https://opencode.ai/config.json",
		"autoupdate": false,
		"share":      "disabled",
		"model":      core.Concat(config.Provider, "/", config.Model),
		"provider": map[string]any{
			config.Provider: map[string]any{
				"npm":  "@ai-sdk/openai-compatible",
				"name": "Core Local",
				"options": map[string]any{
					"apiKey":  "sk-local",
					"baseURL": config.BaseURL,
				},
				"models": models,
			},
		},
		"tools": map[string]any{
			"bash": true,
			"edit": true,
			"glob": true,
			"grep": true,
			"lsp":  true,
			"read": true,
		},
		"permission": map[string]any{
			"bash": "allow",
			"edit": "allow",
			"read": "allow",
		},
	}

	if config.SmallModel != "" {
		content["small_model"] = core.Concat(config.Provider, "/", config.SmallModel)
	}

	return core.JSONMarshalString(content)
}

func opencodeProfileEnvName(profile string) string {
	name := core.Upper(core.Trim(profile))
	name = core.Replace(name, "-", "_")
	name = core.Replace(name, ".", "_")
	name = core.Replace(name, "/", "_")
	return name
}
