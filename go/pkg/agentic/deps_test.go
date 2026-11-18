// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "github.com/dappcore/go"
)

func TestDeps_ParseCoreDeps_Good_Case(t *testing.T) {
	goMod := `module github.com/dappcore/agent/go

go 1.26.0

require (
	github.com/dappcore/go v0.8.0
	github.com/dappcore/go-process/go v0.3.0
	github.com/dappcore/mcp/go v0.4.0
)`

	deps := parseCoreDeps(goMod)

	core.AssertEqual(t, []coreDep{
		{module: "github.com/dappcore/go", repo: "go", dir: "core-go"},
		{module: "github.com/dappcore/go-process/go", repo: "go-process", dir: "core-go-process"},
		{module: "github.com/dappcore/mcp/go", repo: "mcp", dir: "core-mcp"},
	}, deps)
}

func TestDeps_ParseCoreDeps_Bad_NoCoreModules(t *testing.T) {
	goMod := `module example.com/app

go 1.26.0

require github.com/stretchr/testify v1.11.1`

	core.AssertEmpty(t, parseCoreDeps(goMod))
}

func TestDeps_ParseCoreDeps_Ugly_DeduplicatesAndSkipsIndirect(t *testing.T) {
	goMod := `module github.com/dappcore/agent/go

go 1.26.0

require (
	github.com/dappcore/go v0.8.0
	github.com/dappcore/go v0.8.0
	github.com/dappcore/go-ws/go v0.2.0 // indirect
	github.com/dappcore/go-process/go v0.3.0
)`

	core.AssertEqual(t, []coreDep{
		{module: "github.com/dappcore/go", repo: "go", dir: "core-go"},
		{module: "github.com/dappcore/go-process/go", repo: "go-process", dir: "core-go-process"},
	}, parseCoreDeps(goMod))
}

func TestDeps_ForgeSSHURL_Good_Case(t *testing.T) {
	url := forgeSSHURL("core", "go-io")
	core.AssertEqual(t, "ssh://git@forge.lthn.ai:2223/core/go-io.git", url)
	core.AssertContains(t, url, "/core/go-io.git")
}

func TestDeps_CloneWorkspaceDeps_Bad_NoGoMod(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	if r := fs.EnsureDir(repoDir); !r.OK {
		t.Fatalf("mkdir repo: %v", r.Value)
	}

	subsystem := &PrepSubsystem{}
	if err := subsystem.cloneWorkspaceDeps(context.Background(), wsDir, repoDir, "core"); err != nil {
		t.Fatalf("clone workspace deps: %v", err)
	}

	core.AssertFalse(t, fs.IsFile(core.JoinPath(wsDir, "go.work")).OK)
}

func TestDeps_CloneWorkspaceDeps_Ugly_NoDirectCoreDeps(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	if r := fs.EnsureDir(repoDir); !r.OK {
		t.Fatalf("mkdir repo: %v", r.Value)
	}

	goMod := `module example.com/app

go 1.26.0

require (
	github.com/dappcore/go-process/go v0.3.0 // indirect
	github.com/stretchr/testify v1.11.1
)`
	if r := fs.Write(core.JoinPath(repoDir, "go.mod"), goMod); !r.OK {
		t.Fatalf("write go.mod: %v", r.Value)
	}

	subsystem := &PrepSubsystem{}
	if err := subsystem.cloneWorkspaceDeps(context.Background(), wsDir, repoDir, "core"); err != nil {
		t.Fatalf("clone workspace deps: %v", err)
	}

	core.AssertFalse(t, fs.IsFile(core.JoinPath(wsDir, "go.work")).OK)
}

func TestDeps_coreDepRepoOf_Good(t *testing.T) {
	core.AssertEqual(t, "go-process", coreDepRepoOf("github.com/dappcore/go-process/go"))
	core.AssertEqual(t, "go", coreDepRepoOf("github.com/dappcore/go"))
	core.AssertEqual(t, "cli", coreDepRepoOf("github.com/dappcore/cli/go/pkg/cli/frame"))
}

func TestDeps_coreDepRepoOf_Bad(t *testing.T) {
	core.AssertEqual(t, "", coreDepRepoOf("example.com/app"))
	core.AssertEqual(t, "", coreDepRepoOf(""))
}

func TestDeps_coreDepRepoOf_Ugly(t *testing.T) {
	// Legacy vanity pins from unmigrated repos still resolve. A mixed-case org
	// is a different module path to Go, so it isn't treated as Core.
	core.AssertEqual(t, "go-process", coreDepRepoOf(core.Concat(legacyCoreDepRoot, "/process")))
	core.AssertEqual(t, "go", coreDepRepoOf(legacyCoreDepRoot))
	core.AssertEqual(t, "", coreDepRepoOf("github.com/dAppCore/go-io/go"))
}
