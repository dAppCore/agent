// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestDeps_ParseCoreDeps_Good(t *testing.T) {
	goMod := `module dappco.re/go/agent

go 1.26.0

require (
	dappco.re/go/core v0.8.0
	dappco.re/go/core/process v0.3.0
	dappco.re/go/mcp v0.4.0
)`

	deps := parseCoreDeps(goMod)

	assert.Equal(t, []coreDep{
		{module: "dappco.re/go/core", repo: "go", dir: "core-go"},
		{module: "dappco.re/go/core/process", repo: "go-process", dir: "core-go-process"},
		{module: "dappco.re/go/mcp", repo: "mcp", dir: "core-mcp"},
	}, deps)
}

func TestDeps_ParseCoreDeps_Bad_NoCoreModules(t *testing.T) {
	goMod := `module example.com/app

go 1.26.0

require github.com/stretchr/testify v1.11.1`

	assert.Empty(t, parseCoreDeps(goMod))
}

func TestDeps_ParseCoreDeps_Ugly_DeduplicatesAndSkipsIndirect(t *testing.T) {
	goMod := `module dappco.re/go/agent

go 1.26.0

require (
	dappco.re/go/core v0.8.0
	dappco.re/go/core v0.8.0
	dappco.re/go/core/ws v0.2.0 // indirect
	dappco.re/go/core/process v0.3.0
)`

	assert.Equal(t, []coreDep{
		{module: "dappco.re/go/core", repo: "go", dir: "core-go"},
		{module: "dappco.re/go/core/process", repo: "go-process", dir: "core-go-process"},
	}, parseCoreDeps(goMod))
}

func TestDeps_ForgeSSHURL_Good(t *testing.T) {
	assert.Equal(t, "ssh://git@forge.lthn.ai:2223/core/go-io.git", forgeSSHURL("core", "go-io"))
}

func TestDeps_CloneWorkspaceDeps_Bad_NoGoMod(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	if r := fs.EnsureDir(repoDir); !r.OK {
		t.Fatalf("mkdir repo: %v", r.Value)
	}

	subsystem := &PrepSubsystem{}
	subsystem.cloneWorkspaceDeps(context.Background(), wsDir, repoDir, "core")

	assert.False(t, fs.IsFile(core.JoinPath(wsDir, "go.work")))
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
	dappco.re/go/core/process v0.3.0 // indirect
	github.com/stretchr/testify v1.11.1
)`
	if r := fs.Write(core.JoinPath(repoDir, "go.mod"), goMod); !r.OK {
		t.Fatalf("write go.mod: %v", r.Value)
	}

	subsystem := &PrepSubsystem{}
	subsystem.cloneWorkspaceDeps(context.Background(), wsDir, repoDir, "core")

	assert.False(t, fs.IsFile(core.JoinPath(wsDir, "go.work")))
}
