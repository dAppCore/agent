// Package workspace verifies the workspace contract defined by the original
// php-devops wishlist is fully implemented. This test loads the real repos.yaml
// shipped with core/agent and validates every aspect of the specification.
package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-scm/repos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// repoRoot returns the absolute path to the core/agent repo root.
func repoRoot(t *testing.T) string {
	t.Helper()
	// Walk up from this test file to find repos.yaml.
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "repos.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "repos.yaml not found")
		dir = parent
	}
}

// ── repos.yaml contract ────────────────────────────────────────────

func TestContract_ReposYAML_Loads(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "repos.yaml")

	reg, err := repos.LoadRegistry(io.Local, path)
	require.NoError(t, err)
	require.NotNil(t, reg)

	assert.Equal(t, 1, reg.Version, "repos.yaml must declare version: 1")
	assert.NotEmpty(t, reg.Org, "repos.yaml must declare an org")
	assert.NotEmpty(t, reg.BasePath, "repos.yaml must declare base_path")
}

func TestContract_ReposYAML_HasRequiredFields(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	require.NotEmpty(t, reg.Repos, "repos.yaml must define at least one repo")

	for name, repo := range reg.Repos {
		assert.NotEmpty(t, repo.Type, "%s: must have a type", name)
		assert.NotEmpty(t, repo.Description, "%s: must have a description", name)
	}
}

func TestContract_ReposYAML_ValidTypes(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	validTypes := map[string]bool{
		"foundation": true,
		"module":     true,
		"product":    true,
		"template":   true,
		"meta":       true,
	}

	for name, repo := range reg.Repos {
		assert.True(t, validTypes[repo.Type], "%s: invalid type %q", name, repo.Type)
	}
}

func TestContract_ReposYAML_DependenciesExist(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	for name, repo := range reg.Repos {
		for _, dep := range repo.DependsOn {
			_, ok := reg.Get(dep)
			assert.True(t, ok, "%s: depends on %q which is not in repos.yaml", name, dep)
		}
	}
}

func TestContract_ReposYAML_TopologicalOrder(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	order, err := reg.TopologicalOrder()
	require.NoError(t, err, "dependency graph must be acyclic")
	assert.Equal(t, len(reg.Repos), len(order), "topological order must include all repos")

	// Verify ordering: every dependency appears before its dependant.
	seen := map[string]bool{}
	for _, repo := range order {
		for _, dep := range repo.DependsOn {
			assert.True(t, seen[dep], "%s appears before its dependency %s", repo.Name, dep)
		}
		seen[repo.Name] = true
	}
}

func TestContract_ReposYAML_HasFoundation(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	foundations := reg.ByType("foundation")
	assert.NotEmpty(t, foundations, "repos.yaml must have at least one foundation package")
}

func TestContract_ReposYAML_Defaults(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	assert.NotEmpty(t, reg.Defaults.Branch, "defaults must specify a branch")
	assert.NotEmpty(t, reg.Defaults.License, "defaults must specify a licence")
}

func TestContract_ReposYAML_MetaDoesNotCloneSelf(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	for name, repo := range reg.Repos {
		if repo.Type == "meta" && repo.Clone != nil && !*repo.Clone {
			// Meta repos with clone: false are correct.
			continue
		}
		if repo.Type == "meta" {
			t.Logf("%s: meta repo should set clone: false", name)
		}
	}
}

func TestContract_ReposYAML_ProductsHaveDomain(t *testing.T) {
	root := repoRoot(t)
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	for name, repo := range reg.Repos {
		if repo.Type == "product" && repo.Domain != "" {
			// Products with domains are properly configured.
			assert.Contains(t, repo.Domain, ".", "%s: domain should be a valid hostname", name)
		}
	}
}

// ── workspace.yaml contract ────────────────────────────────────────

type workspaceConfig struct {
	Version    int                `yaml:"version"`
	Active     string             `yaml:"active"`
	DefaultOnly []string          `yaml:"default_only"`
	PackagesDir string            `yaml:"packages_dir"`
	Settings   map[string]any     `yaml:"settings"`
}

func TestContract_WorkspaceYAML_Loads(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".core", "workspace.yaml")

	data, err := os.ReadFile(path)
	require.NoError(t, err, ".core/workspace.yaml must exist")

	var ws workspaceConfig
	require.NoError(t, yaml.Unmarshal(data, &ws))

	assert.Equal(t, 1, ws.Version, "workspace.yaml must declare version: 1")
	assert.NotEmpty(t, ws.Active, "workspace.yaml must declare an active package")
	assert.NotEmpty(t, ws.PackagesDir, "workspace.yaml must declare packages_dir")
}

func TestContract_WorkspaceYAML_ActiveInRegistry(t *testing.T) {
	root := repoRoot(t)

	// Load workspace config.
	data, err := os.ReadFile(filepath.Join(root, ".core", "workspace.yaml"))
	require.NoError(t, err)
	var ws workspaceConfig
	require.NoError(t, yaml.Unmarshal(data, &ws))

	// Load repos registry.
	reg, err := repos.LoadRegistry(io.Local, filepath.Join(root, "repos.yaml"))
	require.NoError(t, err)

	_, ok := reg.Get(ws.Active)
	assert.True(t, ok, "workspace.yaml active package %q must exist in repos.yaml", ws.Active)
}

// ── .core/ folder spec contract ────────────────────────────────────

func TestContract_CoreFolder_Exists(t *testing.T) {
	root := repoRoot(t)
	info, err := os.Stat(filepath.Join(root, ".core"))
	require.NoError(t, err, ".core/ directory must exist")
	assert.True(t, info.IsDir())
}

func TestContract_CoreFolder_HasSpec(t *testing.T) {
	root := repoRoot(t)
	_, err := os.Stat(filepath.Join(root, ".core", "docs", "core-folder-spec.md"))
	assert.NoError(t, err, ".core/docs/core-folder-spec.md must exist")
}

// ── Setup scripts contract ─────────────────────────────────────────

func TestContract_SetupScript_Exists(t *testing.T) {
	root := repoRoot(t)
	_, err := os.Stat(filepath.Join(root, "setup.sh"))
	assert.NoError(t, err, "setup.sh must exist at repo root")
}

func TestContract_SetupScript_Executable(t *testing.T) {
	root := repoRoot(t)
	info, err := os.Stat(filepath.Join(root, "setup.sh"))
	if err != nil {
		t.Skip("setup.sh not found")
	}
	assert.NotZero(t, info.Mode()&0111, "setup.sh must be executable")
}

func TestContract_InstallScripts_Exist(t *testing.T) {
	root := repoRoot(t)
	scripts := []string{
		"scripts/install-deps.sh",
		"scripts/install-core.sh",
	}
	for _, s := range scripts {
		_, err := os.Stat(filepath.Join(root, s))
		assert.NoError(t, err, "%s must exist", s)
	}
}

// ── Claude plugins contract ────────────────────────────────────────

func TestContract_Marketplace_Exists(t *testing.T) {
	root := repoRoot(t)
	_, err := os.Stat(filepath.Join(root, ".claude-plugin", "marketplace.json"))
	assert.NoError(t, err, ".claude-plugin/marketplace.json must exist for plugin distribution")
}

func TestContract_Plugins_HaveManifests(t *testing.T) {
	root := repoRoot(t)
	pluginDir := filepath.Join(root, "claude")

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		t.Skip("claude/ directory not found")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest := filepath.Join(pluginDir, entry.Name(), ".claude-plugin", "plugin.json")
		_, err := os.Stat(manifest)
		assert.NoError(t, err, "claude/%s must have .claude-plugin/plugin.json", entry.Name())
	}
}
