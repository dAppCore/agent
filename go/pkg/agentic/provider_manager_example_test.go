// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

func ExampleNewProviderManager() {
	manager := NewProviderManager(nil)
	core.Println(len(manager.Names()))
	// Output: 3
}

func ExampleProvider_Generate() {
	provider := newContentProvider("claude", "model", true, func(context.Context, string, map[string]any) (string, error) {
		return "draft", nil
	})
	text, _ := provider.Generate(context.Background(), "Write a release note", nil)
	core.Println(text)
	// Output: draft
}

func ExampleProvider_Stream() {
	provider := newContentProvider("claude", "model", true, func(context.Context, string, map[string]any) (string, error) {
		return "draft", nil
	})
	streamed := ""
	_ = provider.Stream(context.Background(), "Write a release note", nil, func(token string) { streamed = token })
	core.Println(streamed)
	// Output: draft
}

func ExampleProvider_Name() {
	provider := newContentProvider("claude", "model", true, nil)
	core.Println(provider.Name())
	// Output: claude
}

func ExampleProvider_DefaultModel() {
	provider := newContentProvider("claude", "gpt-5.4", true, nil)
	core.Println(provider.DefaultModel())
	// Output: gpt-5.4
}

func ExampleProvider_IsAvailable() {
	provider := newContentProvider("claude", "model", true, nil)
	core.Println(provider.IsAvailable())
	// Output: true
}

func ExampleProviderManager_Register() {
	manager := &ProviderManager{}
	manager.Register(newContentProvider("custom", "gpt-5.4", true, nil))
	core.Println(manager.Names()[0])
	// Output: custom
}

func ExampleProviderManager_Provider() {
	manager := NewProviderManager(nil)
	provider, ok := manager.Provider("claude")
	core.Println(ok)
	core.Println(provider.Name())
	// Output:
	// true
	// claude
}

func ExampleProviderManager_Names() {
	core.Println(NewProviderManager(nil).Names())
	// Output: [claude gemini openai]
}

func ExampleProviderManager_DefaultProvider() {
	manager := &ProviderManager{}
	manager.Register(newContentProvider("custom", "gpt-5.4", true, nil))
	core.Println(manager.DefaultProvider().Name())
	// Output: custom
}
