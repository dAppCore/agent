// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestOpencodeProvider_NewProviderManager_Good_RegistersOpencode(t *testing.T) {
	manager := newOpencodeProviderManager(core.New())

	provider, ok := manager.Provider(opencodeProviderName)
	core.AssertTrue(t, ok, "opencode provider should be registered")
	core.AssertEqual(t, opencodeProviderName, provider.Name())
	core.AssertEqual(t, opencodeDefaultModel, provider.DefaultModel())
	core.AssertTrue(t, provider.IsAvailable(), "opencode provider should report available")

	// The named providers are real (opencode-backed), not nil-generate.
	for _, name := range []string{"claude", "gemini", "openai"} {
		p, found := manager.Provider(name)
		core.AssertTrue(t, found, "named provider should still register: "+name)
		core.AssertTrue(t, p.IsAvailable(), "named provider should be available: "+name)
	}
}

func TestOpencodeProvider_Generate_Bad_ServiceNotRegistered(t *testing.T) {
	// core.New() has no opencode service — Generate must fail loud with a
	// clear error rather than the old nil-generate "provider not configured".
	generate := newOpencodeGenerate(core.New())

	_, err := generate(context.Background(), "hello", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "opencode service not registered")
}

func TestOpencodeProvider_Generate_Bad_NilCore(t *testing.T) {
	generate := newOpencodeGenerate(nil)

	_, err := generate(context.Background(), "hello", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "core unavailable")
}

func TestOpencodeProvider_opencodeMessageModel_Good(t *testing.T) {
	// A caller-pinned provider/model form passes through unchanged.
	core.AssertEqual(t, "core-local/lthn/lemma",
		opencodeMessageModel(map[string]any{"model": "core-local/lthn/lemma"}))
}

func TestOpencodeProvider_opencodeMessageModel_Ugly_DropsProfileSentinel(t *testing.T) {
	// The ProviderManager wrapper injects the default-model sentinel
	// (a PROFILE name) when the caller pins nothing — it must be dropped
	// so opencode-serve uses the profile's configured model.
	core.AssertEqual(t, "",
		opencodeMessageModel(map[string]any{"model": opencodeDefaultModel}))
	core.AssertEqual(t, "", opencodeMessageModel(nil))
}

func TestOpencodeProvider_optionMapString_Good(t *testing.T) {
	options := map[string]any{"profile": "lemma", "sandbox-id": "oc-9"}

	core.AssertEqual(t, "lemma", optionMapString(options, "profile"))
	// First non-empty across alias keys wins.
	core.AssertEqual(t, "oc-9", optionMapString(options, "sandbox_id", "sandbox-id"))
}

func TestOpencodeProvider_optionMapString_Bad_MissingAndWrongType(t *testing.T) {
	options := map[string]any{"profile": 42, "agent": "   "}

	core.AssertEqual(t, "", optionMapString(options, "missing"))
	// Non-string value is ignored.
	core.AssertEqual(t, "", optionMapString(options, "profile"))
	// Whitespace-only is treated as empty.
	core.AssertEqual(t, "", optionMapString(options, "agent"))
}
