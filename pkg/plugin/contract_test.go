// Package plugin verifies the Claude Code plugin contract. Every plugin in the
// marketplace must satisfy the structural rules Claude Code expects: valid JSON
// manifests, commands with YAML frontmatter, executable scripts, and well-formed
// hooks. These tests catch breakage before a tag ships.
package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── types ──────────────────────────────────────────────────────────

type marketplace struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Owner       owner    `json:"owner"`
	Plugins     []plugin `json:"plugins"`
}

type owner struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type plugin struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type pluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type hooksFile struct {
	Schema string                     `json:"$schema"`
	Hooks  map[string]json.RawMessage `json:"hooks"`
}

type hookEntry struct {
	Matcher     string     `json:"matcher"`
	Hooks       []hookDef  `json:"hooks"`
	Description string     `json:"description"`
}

type hookDef struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// ── helpers ────────────────────────────────────────────────────────

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".claude-plugin", "marketplace.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "marketplace.json not found")
		dir = parent
	}
}

func loadMarketplace(t *testing.T) (marketplace, string) {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".claude-plugin", "marketplace.json"))
	require.NoError(t, err)
	var mp marketplace
	require.NoError(t, json.Unmarshal(data, &mp))
	return mp, root
}

// validHookEvents are the hook events Claude Code supports.
var validHookEvents = map[string]bool{
	"PreToolUse":        true,
	"PostToolUse":       true,
	"Stop":              true,
	"SubagentStop":      true,
	"SessionStart":      true,
	"SessionEnd":        true,
	"UserPromptSubmit":  true,
	"PreCompact":        true,
	"Notification":      true,
}

// ── Marketplace contract ───────────────────────────────────────────

func TestMarketplace_Valid(t *testing.T) {
	mp, _ := loadMarketplace(t)
	assert.NotEmpty(t, mp.Name, "marketplace must have a name")
	assert.NotEmpty(t, mp.Description, "marketplace must have a description")
	assert.NotEmpty(t, mp.Owner.Name, "marketplace must have an owner name")
	assert.NotEmpty(t, mp.Plugins, "marketplace must list at least one plugin")
}

func TestMarketplace_PluginsHaveRequiredFields(t *testing.T) {
	mp, _ := loadMarketplace(t)
	for _, p := range mp.Plugins {
		assert.NotEmpty(t, p.Name, "plugin must have a name")
		assert.NotEmpty(t, p.Source, "plugin %s must have a source path", p.Name)
		assert.NotEmpty(t, p.Description, "plugin %s must have a description", p.Name)
		assert.NotEmpty(t, p.Version, "plugin %s must have a version", p.Name)
	}
}

func TestMarketplace_UniquePluginNames(t *testing.T) {
	mp, _ := loadMarketplace(t)
	seen := map[string]bool{}
	for _, p := range mp.Plugins {
		assert.False(t, seen[p.Name], "duplicate plugin name: %s", p.Name)
		seen[p.Name] = true
	}
}

// ── Plugin directory structure ─────────────────────────────────────

func TestPlugin_DirectoryExists(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		pluginDir := filepath.Join(root, p.Source)
		info, err := os.Stat(pluginDir)
		require.NoError(t, err, "plugin %s: source dir %s must exist", p.Name, p.Source)
		assert.True(t, info.IsDir(), "plugin %s: source must be a directory", p.Name)
	}
}

func TestPlugin_HasManifest(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		manifestPath := filepath.Join(root, p.Source, ".claude-plugin", "plugin.json")
		data, err := os.ReadFile(manifestPath)
		require.NoError(t, err, "plugin %s: must have .claude-plugin/plugin.json", p.Name)

		var manifest pluginManifest
		require.NoError(t, json.Unmarshal(data, &manifest), "plugin %s: invalid plugin.json", p.Name)
		assert.NotEmpty(t, manifest.Name, "plugin %s: manifest must have a name", p.Name)
		assert.NotEmpty(t, manifest.Description, "plugin %s: manifest must have a description", p.Name)
		assert.NotEmpty(t, manifest.Version, "plugin %s: manifest must have a version", p.Name)
	}
}

// ── Commands contract ──────────────────────────────────────────────

func TestPlugin_CommandsAreMarkdown(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		cmdDir := filepath.Join(root, p.Source, "commands")
		entries, err := os.ReadDir(cmdDir)
		if os.IsNotExist(err) {
			continue // commands dir is optional
		}
		require.NoError(t, err, "plugin %s: failed to read commands dir", p.Name)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			assert.True(t, strings.HasSuffix(entry.Name(), ".md"),
				"plugin %s: command %s must be a .md file", p.Name, entry.Name())
		}
	}
}

func TestPlugin_CommandsHaveFrontmatter(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		cmdDir := filepath.Join(root, p.Source, "commands")
		entries, err := os.ReadDir(cmdDir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(cmdDir, entry.Name()))
			require.NoError(t, err)

			content := string(data)
			assert.True(t, strings.HasPrefix(content, "---"),
				"plugin %s: command %s must start with YAML frontmatter (---)", p.Name, entry.Name())

			// Must have closing frontmatter
			parts := strings.SplitN(content[3:], "---", 2)
			assert.True(t, len(parts) >= 2,
				"plugin %s: command %s must have closing frontmatter (---)", p.Name, entry.Name())

			// Frontmatter must contain name:
			assert.Contains(t, parts[0], "name:",
				"plugin %s: command %s frontmatter must contain 'name:'", p.Name, entry.Name())
		}
	}
}

// ── Hooks contract ─────────────────────────────────────────────────

func TestPlugin_HooksFileValid(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		hooksPath := filepath.Join(root, p.Source, "hooks.json")
		data, err := os.ReadFile(hooksPath)
		if os.IsNotExist(err) {
			continue // hooks.json is optional
		}
		require.NoError(t, err, "plugin %s: failed to read hooks.json", p.Name)

		var hf hooksFile
		require.NoError(t, json.Unmarshal(data, &hf), "plugin %s: invalid hooks.json", p.Name)
		assert.NotEmpty(t, hf.Hooks, "plugin %s: hooks.json must define at least one event", p.Name)
	}
}

func TestPlugin_HooksUseValidEvents(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		hooksPath := filepath.Join(root, p.Source, "hooks.json")
		data, err := os.ReadFile(hooksPath)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		var hf hooksFile
		require.NoError(t, json.Unmarshal(data, &hf))

		for event := range hf.Hooks {
			assert.True(t, validHookEvents[event],
				"plugin %s: unknown hook event %q (valid: %v)", p.Name, event, validHookEvents)
		}
	}
}

func TestPlugin_HookScriptsExist(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		hooksPath := filepath.Join(root, p.Source, "hooks.json")
		data, err := os.ReadFile(hooksPath)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		var hf hooksFile
		require.NoError(t, json.Unmarshal(data, &hf))

		pluginRoot := filepath.Join(root, p.Source)

		for event, raw := range hf.Hooks {
			var entries []hookEntry
			require.NoError(t, json.Unmarshal(raw, &entries),
				"plugin %s: failed to parse %s entries", p.Name, event)

			for _, entry := range entries {
				for _, h := range entry.Hooks {
					if h.Type != "command" {
						continue
					}
					// Resolve ${CLAUDE_PLUGIN_ROOT} to the plugin source directory
					cmd := strings.ReplaceAll(h.Command, "${CLAUDE_PLUGIN_ROOT}", pluginRoot)
					// Extract the script path (first arg, before any flags)
					scriptPath := strings.Fields(cmd)[0]
					_, err := os.Stat(scriptPath)
					assert.NoError(t, err,
						"plugin %s: hook script %s does not exist (event: %s)", p.Name, h.Command, event)
				}
			}
		}
	}
}

func TestPlugin_HookScriptsExecutable(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		hooksPath := filepath.Join(root, p.Source, "hooks.json")
		data, err := os.ReadFile(hooksPath)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		var hf hooksFile
		require.NoError(t, json.Unmarshal(data, &hf))

		pluginRoot := filepath.Join(root, p.Source)

		for event, raw := range hf.Hooks {
			var entries []hookEntry
			require.NoError(t, json.Unmarshal(raw, &entries))

			for _, entry := range entries {
				for _, h := range entry.Hooks {
					if h.Type != "command" {
						continue
					}
					cmd := strings.ReplaceAll(h.Command, "${CLAUDE_PLUGIN_ROOT}", pluginRoot)
					scriptPath := strings.Fields(cmd)[0]
					info, err := os.Stat(scriptPath)
					if err != nil {
						continue // Already caught by ScriptsExist test
					}
					assert.NotZero(t, info.Mode()&0111,
						"plugin %s: hook script %s must be executable (event: %s)", p.Name, h.Command, event)
				}
			}
		}
	}
}

// ── Scripts contract ───────────────────────────────────────────────

func TestPlugin_AllScriptsExecutable(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		scriptsDir := filepath.Join(root, p.Source, "scripts")
		entries, err := os.ReadDir(scriptsDir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".sh") {
				continue
			}
			info, err := entry.Info()
			require.NoError(t, err)
			assert.NotZero(t, info.Mode()&0111,
				"plugin %s: script %s must be executable", p.Name, entry.Name())
		}
	}
}

func TestPlugin_ScriptsHaveShebang(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		scriptsDir := filepath.Join(root, p.Source, "scripts")
		entries, err := os.ReadDir(scriptsDir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(scriptsDir, entry.Name()))
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(string(data), "#!"),
				"plugin %s: script %s must start with a shebang (#!)", p.Name, entry.Name())
		}
	}
}

// ── Skills contract ────────────────────────────────────────────────

func TestPlugin_SkillsHaveSkillMd(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		skillsDir := filepath.Join(root, p.Source, "skills")
		entries, err := os.ReadDir(skillsDir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillMd := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			_, err := os.Stat(skillMd)
			assert.NoError(t, err,
				"plugin %s: skill %s must have a SKILL.md", p.Name, entry.Name())
		}
	}
}

func TestPlugin_SkillScriptsExecutable(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		skillsDir := filepath.Join(root, p.Source, "skills")
		entries, err := os.ReadDir(skillsDir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDir := filepath.Join(skillsDir, entry.Name())
			scripts, _ := os.ReadDir(skillDir)
			for _, s := range scripts {
				if s.IsDir() || !strings.HasSuffix(s.Name(), ".sh") {
					continue
				}
				info, err := s.Info()
				require.NoError(t, err)
				assert.NotZero(t, info.Mode()&0111,
					"plugin %s: skill script %s/%s must be executable", p.Name, entry.Name(), s.Name())
			}
		}
	}
}

// ── Cross-references ───────────────────────────────────────────────

func TestPlugin_CollectionScriptsExecutable(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		collDir := filepath.Join(root, p.Source, "collection")
		entries, err := os.ReadDir(collDir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
				continue
			}
			info, err := entry.Info()
			require.NoError(t, err)
			assert.NotZero(t, info.Mode()&0111,
				"plugin %s: collection script %s must be executable", p.Name, entry.Name())
		}
	}
}

func TestMarketplace_SourcesMatchDirectories(t *testing.T) {
	mp, root := loadMarketplace(t)

	// Every directory in claude/ should be listed in marketplace
	claudeDir := filepath.Join(root, "claude")
	entries, err := os.ReadDir(claudeDir)
	require.NoError(t, err)

	pluginNames := map[string]bool{}
	for _, p := range mp.Plugins {
		pluginNames[p.Name] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		assert.True(t, pluginNames[entry.Name()],
			"directory claude/%s exists but is not listed in marketplace.json", entry.Name())
	}
}

func TestMarketplace_VersionConsistency(t *testing.T) {
	mp, root := loadMarketplace(t)
	for _, p := range mp.Plugins {
		manifestPath := filepath.Join(root, p.Source, ".claude-plugin", "plugin.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // Already caught by HasManifest test
		}
		var manifest pluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		assert.Equal(t, p.Version, manifest.Version,
			"plugin %s: marketplace version %q != manifest version %q", p.Name, p.Version, manifest.Version)
	}
}
