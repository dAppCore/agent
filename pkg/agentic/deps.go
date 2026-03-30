// SPDX-License-Identifier: EUPL-1.2

// Workspace dependency cloning.
// Reads the repo's go.mod, finds Core ecosystem modules, clones them into
// the workspace alongside ./repo, and builds a go.work that includes them all.
// This gives the agent a complete, buildable workspace without needing go.work=off.

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

// s.cloneWorkspaceDeps(ctx, workspaceDir, repoDir, "core")
func (s *PrepSubsystem) cloneWorkspaceDeps(ctx context.Context, workspaceDir, repoDir, org string) {
	goModPath := core.JoinPath(repoDir, "go.mod")
	r := fs.Read(goModPath)
	if !r.OK {
		return // no go.mod — not a Go project
	}
	// Parse requires from go.mod
	deps := parseCoreDeps(r.Value.(string))
	if len(deps) == 0 {
		return
	}
	if s.ServiceRuntime == nil {
		return
	}
	process := s.Core().Process()

	// Deduplicate (dappco.re and forge.lthn.ai may map to same repo)
	dedupSeen := make(map[string]bool)
	var unique []coreDep
	for _, dep := range deps {
		if !dedupSeen[dep.dir] {
			dedupSeen[dep.dir] = true
			unique = append(unique, dep)
		}
	}
	deps = unique

	// Clone each dependency
	var cloned []string
	for _, dep := range deps {
		depDir := core.JoinPath(workspaceDir, dep.dir)
		if fs.IsDir(core.JoinPath(depDir, ".git")) {
			cloned = append(cloned, dep.dir) // already cloned (resume)
			continue
		}

		repoURL := forgeSSHURL(org, dep.repo)
		if result := process.RunIn(ctx, workspaceDir, "git", "clone", "--depth=1", repoURL, dep.dir); result.OK {
			cloned = append(cloned, dep.dir)
		}
	}

	// Rebuild go.work with repo + all cloned deps
	if len(cloned) > 0 {
		b := core.NewBuilder()
		b.WriteString("go 1.26.0\n\nuse (\n")
		b.WriteString("\t./repo\n")
		for _, dir := range cloned {
			b.WriteString(core.Concat("\t./", dir, "\n"))
		}
		b.WriteString(")\n")
		fs.Write(core.JoinPath(workspaceDir, "go.work"), b.String())
	}
}

// coreDep maps a Go module path to a Forge repo clone target.
type coreDep struct {
	module string // e.g. "dappco.re/go/core"
	repo   string // e.g. "go" (Forge repo name)
	dir    string // e.g. "core-go" (workspace subdir)
}

// deps := parseCoreDeps(goMod)
// if len(deps) > 0 { core.Println(deps[0].repo) }
func parseCoreDeps(gomod string) []coreDep {
	var deps []coreDep
	seen := make(map[string]bool)

	for _, line := range core.Split(gomod, "\n") {
		line = core.Trim(line)

		// Skip indirect dependencies
		if core.Contains(line, "// indirect") {
			continue
		}

		// Match dappco.re/go/* requires
		if core.HasPrefix(line, "dappco.re/go/") {
			parts := core.Split(line, " ")
			mod := parts[0]
			if seen[mod] {
				continue
			}
			seen[mod] = true

			// dappco.re/go/core → repo "go", dir "core-go"
			// dappco.re/go/core/process → repo "go-process", dir "core-go-process"
			// dappco.re/go/core/ws → repo "go-ws", dir "core-go-ws"
			// dappco.re/go/mcp → repo "mcp", dir "core-mcp"
			suffix := core.TrimPrefix(mod, "dappco.re/go/")
			repo := suffix
			if core.HasPrefix(suffix, "core/") {
				// core/process → go-process
				repoSuffix := core.TrimPrefix(suffix, "core/")
				repo = core.Concat("go-", repoSuffix)
			} else if suffix == "core" {
				repo = "go"
			}
			dir := core.Concat("core-", core.Replace(repo, "/", "-"))
			deps = append(deps, coreDep{module: mod, repo: repo, dir: dir})
		}

		// Match forge.lthn.ai/core/* requires (legacy paths)
		//if core.HasPrefix(line, "forge.lthn.ai/core/") {
		//	parts := core.Split(line, " ")
		//	mod := parts[0]
		//	if seen[mod] {
		//		continue
		//	}
		//	seen[mod] = true
		//
		//	suffix := core.TrimPrefix(mod, "forge.lthn.ai/core/")
		//	repo := suffix
		//	dir := core.Concat("core-", core.Replace(repo, "/", "-"))
		//	deps = append(deps, coreDep{module: mod, repo: repo, dir: dir})
		//}
	}

	return deps
}

// forgeSSHURL builds the Forge SSH clone URL for a repo.
//
//	forgeSSHURL("core", "go-io") → "ssh://git@forge.lthn.ai:2223/core/go-io.git"
func forgeSSHURL(org, repo string) string {
	return core.Concat("ssh://git@forge.lthn.ai:2223/", org, "/", repo, ".git")
}
