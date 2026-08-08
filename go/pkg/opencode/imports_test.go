// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"
	"time"
)

// --- SourceOpenCodeHost ---

// TestSourceOpenCodeHost_Value_Ugly — the constant must be the expected
// string so query filters in the orm layer are stable.
func TestSourceOpenCodeHost_Value_Ugly(t *testing.T) {
	if SourceOpenCodeHost != "opencode-host" {
		t.Errorf("SourceOpenCodeHost = %q; want opencode-host", SourceOpenCodeHost)
	}
}

// --- ImportedProject ---

// TestImportedProject_DefaultZeroValue_Ugly — a zero-value ImportedProject
// must have empty string fields and zero timestamps.
func TestImportedProject_DefaultZeroValue_Ugly(t *testing.T) {
	var p ImportedProject
	if p.ID != "" {
		t.Errorf("zero ImportedProject.ID = %q; want empty", p.ID)
	}
	if p.Source != "" {
		t.Errorf("zero ImportedProject.Source = %q; want empty", p.Source)
	}
	if p.SourceID != "" {
		t.Errorf("zero ImportedProject.SourceID = %q; want empty", p.SourceID)
	}
	if p.Name != "" {
		t.Errorf("zero ImportedProject.Name = %q; want empty", p.Name)
	}
	if p.Worktree != "" {
		t.Errorf("zero ImportedProject.Worktree = %q; want empty", p.Worktree)
	}
	if p.VCS != "" {
		t.Errorf("zero ImportedProject.VCS = %q; want empty", p.VCS)
	}
}

// TestImportedProject_FieldAssignment_Good — all fields of ImportedProject
// must be settable and retrievable.
func TestImportedProject_FieldAssignment_Good(t *testing.T) {
	now := time.Now()
	p := ImportedProject{
		ID:                "opencode-host:abc123",
		Source:            SourceOpenCodeHost,
		SourceID:          "abc123",
		Name:              "my-project",
		Worktree:          "/home/user/projects/my-project",
		VCS:               "git",
		IconColor:         "purple",
		IconDataURL:       "data:image/png;base64,...",
		SandboxesJSON:     `["child-1","child-2"]`,
		UpstreamCreatedAt: now,
		UpstreamUpdatedAt: now,
		ImportedAt:        now,
	}
	if p.ID != "opencode-host:abc123" {
		t.Errorf("ID = %q; want opencode-host:abc123", p.ID)
	}
	if p.Source != SourceOpenCodeHost {
		t.Errorf("Source = %q; want %q", p.Source, SourceOpenCodeHost)
	}
	if p.SourceID != "abc123" {
		t.Errorf("SourceID = %q; want abc123", p.SourceID)
	}
	if p.Name != "my-project" {
		t.Errorf("Name = %q; want my-project", p.Name)
	}
	if p.Worktree != "/home/user/projects/my-project" {
		t.Errorf("Worktree = %q", p.Worktree)
	}
	if p.VCS != "git" {
		t.Errorf("VCS = %q; want git", p.VCS)
	}
	if p.IconColor != "purple" {
		t.Errorf("IconColor = %q; want purple", p.IconColor)
	}
	if !p.UpstreamCreatedAt.Equal(now) {
		t.Errorf("UpstreamCreatedAt = %v; want %v", p.UpstreamCreatedAt, now)
	}
	if !p.UpstreamUpdatedAt.Equal(now) {
		t.Errorf("UpstreamUpdatedAt = %v; want %v", p.UpstreamUpdatedAt, now)
	}
	if !p.ImportedAt.Equal(now) {
		t.Errorf("ImportedAt = %v; want %v", p.ImportedAt, now)
	}
}

// TestImportedProject_SchemaReturnsOrmDefinition_Good — Schema must
// return a non-nil orm schema with the expected table name.
func TestImportedProject_SchemaReturnsOrmDefinition_Good(t *testing.T) {
	schema := ImportedProject{}.Schema()
	if schema.Name != "imported_projects" {
		t.Errorf("schema.Name = %q; want 'imported_projects'", schema.Name)
	}
	if len(schema.PK) == 0 {
		t.Error("schema.PK must not be empty")
	}
}

// TestImportedProject_SchemaHasExpectedFields_Good — the schema must
// declare the core routing fields.
func TestImportedProject_SchemaHasExpectedFields_Good(t *testing.T) {
	schema := ImportedProject{}.Schema()
	fields := map[string]bool{}
	for _, f := range schema.Fields {
		fields[f.Name] = true
	}
	for _, name := range []string{"id", "source", "source_id", "name", "worktree", "imported_at"} {
		if !fields[name] {
			t.Errorf("schema missing expected field %q", name)
		}
	}
}

// --- ImportedProvider ---

// TestImportedProvider_DefaultZeroValue_Ugly — a zero-value
// ImportedProvider must have empty string fields and HasAuth false.
func TestImportedProvider_DefaultZeroValue_Ugly(t *testing.T) {
	var p ImportedProvider
	if p.ID != "" {
		t.Errorf("zero ImportedProvider.ID = %q; want empty", p.ID)
	}
	if p.Source != "" {
		t.Errorf("zero ImportedProvider.Source = %q; want empty", p.Source)
	}
	if p.ProviderID != "" {
		t.Errorf("zero ImportedProvider.ProviderID = %q; want empty", p.ProviderID)
	}
	if p.HasAuth {
		t.Errorf("zero ImportedProvider.HasAuth = true; want false")
	}
}

// TestImportedProvider_FieldAssignment_Good — all fields must be
// settable and retrievable.
func TestImportedProvider_FieldAssignment_Good(t *testing.T) {
	now := time.Now()
	p := ImportedProvider{
		ID:          "opencode-host:anthropic",
		Source:      SourceOpenCodeHost,
		ProviderID:  "anthropic",
		Name:        "Anthropic",
		NPM:         "@ai-sdk/anthropic",
		OptionsJSON: `{"baseURL":"https://api.anthropic.com/v1"}`,
		AuthType:    "apikey",
		AuthKey:     "sk-ant-...",
		HasAuth:     true,
		ImportedAt:  now,
	}
	if p.ID != "opencode-host:anthropic" {
		t.Errorf("ID = %q; want opencode-host:anthropic", p.ID)
	}
	if p.Source != SourceOpenCodeHost {
		t.Errorf("Source = %q; want %q", p.Source, SourceOpenCodeHost)
	}
	if p.ProviderID != "anthropic" {
		t.Errorf("ProviderID = %q; want anthropic", p.ProviderID)
	}
	if p.Name != "Anthropic" {
		t.Errorf("Name = %q; want Anthropic", p.Name)
	}
	if p.NPM != "@ai-sdk/anthropic" {
		t.Errorf("NPM = %q; want @ai-sdk/anthropic", p.NPM)
	}
	if p.AuthType != "apikey" {
		t.Errorf("AuthType = %q; want apikey", p.AuthType)
	}
	if p.AuthKey != "sk-ant-..." {
		t.Errorf("AuthKey = %q; want sk-ant-...", p.AuthKey)
	}
	if !p.HasAuth {
		t.Errorf("HasAuth = false; want true")
	}
	if !p.ImportedAt.Equal(now) {
		t.Errorf("ImportedAt = %v; want %v", p.ImportedAt, now)
	}
}

// TestImportedProvider_NoAuth_Good — a provider without auth must have
// HasAuth = false and empty AuthKey.
func TestImportedProvider_NoAuth_Good(t *testing.T) {
	p := ImportedProvider{
		ID:         "opencode-host:openai",
		Source:     SourceOpenCodeHost,
		ProviderID: "openai",
		Name:       "OpenAI",
		HasAuth:    false,
	}
	if p.HasAuth {
		t.Error("expected HasAuth = false for no-auth provider")
	}
	if p.AuthKey != "" {
		t.Errorf("AuthKey = %q; want empty", p.AuthKey)
	}
	if p.AuthType != "" {
		t.Errorf("AuthType = %q; want empty", p.AuthType)
	}
}

// TestImportedProvider_SchemaReturnsOrmDefinition_Good — Schema must
// return a non-nil orm schema with the expected table name.
func TestImportedProvider_SchemaReturnsOrmDefinition_Good(t *testing.T) {
	schema := ImportedProvider{}.Schema()
	if schema.Name != "imported_providers" {
		t.Errorf("schema.Name = %q; want 'imported_providers'", schema.Name)
	}
	if len(schema.PK) == 0 {
		t.Error("schema.PK must not be empty")
	}
}

// TestImportedProvider_SchemaHasExpectedFields_Good — the schema must
// declare the core routing fields.
func TestImportedProvider_SchemaHasExpectedFields_Good(t *testing.T) {
	schema := ImportedProvider{}.Schema()
	fields := map[string]bool{}
	for _, f := range schema.Fields {
		fields[f.Name] = true
	}
	for _, name := range []string{"id", "source", "provider_id", "name", "has_auth", "imported_at"} {
		if !fields[name] {
			t.Errorf("schema missing expected field %q", name)
		}
	}
}
