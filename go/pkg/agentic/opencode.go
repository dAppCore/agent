// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

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
	config := opencodeProfileConfig(profile)
	model := core.Concat(config.Provider, "/", config.Model)

	builder := core.NewBuilder()
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
