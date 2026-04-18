// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

// s.cloneWorkspaceDeps(ctx, workspaceDir, repoDir, "core")
func (s *PrepSubsystem) cloneWorkspaceDeps(ctx context.Context, workspaceDir, repoDir, org string) error {
	goModPath := core.JoinPath(repoDir, "go.mod")
	r := fs.Read(goModPath)
	if !r.OK {
		return nil
	}
	deps := parseCoreDeps(r.Value.(string))
	if len(deps) == 0 {
		return nil
	}
	if s.ServiceRuntime == nil {
		return nil
	}
	process := s.Core().Process()

	dedupSeen := make(map[string]bool)
	var unique []coreDep
	for _, dep := range deps {
		if !dedupSeen[dep.dir] {
			dedupSeen[dep.dir] = true
			unique = append(unique, dep)
		}
	}
	deps = unique

	var cloned []string
	for _, dep := range deps {
		depDir := core.JoinPath(workspaceDir, dep.dir)
		if fs.IsDir(core.JoinPath(depDir, ".git")) {
			cloned = append(cloned, dep.dir)
			continue
		}

		repoURL := forgeSSHURL(org, dep.repo)
		if result := process.RunIn(ctx, workspaceDir, "git", "clone", "--depth=1", repoURL, dep.dir); result.OK {
			cloned = append(cloned, dep.dir)
		}
	}

	if len(cloned) > 0 {
		b := core.NewBuilder()
		b.WriteString("go 1.26.0\n\nuse (\n")
		b.WriteString("\t./repo\n")
		for _, dir := range cloned {
			b.WriteString(core.Concat("\t./", dir, "\n"))
		}
		b.WriteString(")\n")
		if r := fs.WriteAtomic(core.JoinPath(workspaceDir, "go.work"), b.String()); !r.OK {
			if err, ok := r.Value.(error); ok {
				return core.E("cloneWorkspaceDeps", "write go.work", err)
			}
			return core.E("cloneWorkspaceDeps", "write go.work", nil)
		}
	}

	return nil
}

// dep := coreDep{module: "dappco.re/go/core", repo: "go", dir: "core-go"}
type coreDep struct {
	module string
	repo   string
	dir    string
}

// deps := parseCoreDeps(goMod)
// if len(deps) > 0 { core.Println(deps[0].repo) }
func parseCoreDeps(gomod string) []coreDep {
	var deps []coreDep
	seen := make(map[string]bool)

	for _, line := range core.Split(gomod, "\n") {
		line = core.Trim(line)

		if core.Contains(line, "// indirect") {
			continue
		}

		if core.HasPrefix(line, "dappco.re/go/") {
			parts := core.Split(line, " ")
			mod := parts[0]
			if seen[mod] {
				continue
			}
			seen[mod] = true

			suffix := core.TrimPrefix(mod, "dappco.re/go/")
			repo := suffix
			if core.HasPrefix(suffix, "core/") {
				repoSuffix := core.TrimPrefix(suffix, "core/")
				repo = core.Concat("go-", repoSuffix)
			} else if suffix == "core" {
				repo = "go"
			}
			dir := core.Concat("core-", core.Replace(repo, "/", "-"))
			deps = append(deps, coreDep{module: mod, repo: repo, dir: dir})
		}
	}

	return deps
}

// url := forgeSSHURL("core", "go-io")
// core.Println(url) // "ssh://git@forge.lthn.ai:2223/core/go-io.git"
func forgeSSHURL(org, repo string) string {
	return core.Concat("ssh://git@forge.lthn.ai:2223/", org, "/", repo, ".git")
}
