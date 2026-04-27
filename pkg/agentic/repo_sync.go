// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// Repo sync closes the RFC §7 loop from a pushed workspace back to the local
// mirror. The 5-minute fallback fetch scheduler already lives in fetch_loop.go;
// this file wires the event-driven refresh and the manual command surface.

// RepoSyncOutput is the per-repo result of a workspace→local mirror sync:
// org/repo/branch identifiers, the local repo dir touched, and whether a
// hard reset was performed. Emitted by the repo-sync command + watcher.
//
//	out := RepoSyncOutput{Success: true, Org: "core", Repo: "go-store", Branch: "dev"}
type RepoSyncOutput struct {
	Success bool   `json:"success"`
	Org     string `json:"org,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Branch  string `json:"branch,omitempty"`
	RepoDir string `json:"repo_dir,omitempty"`
	Reset   bool   `json:"reset,omitempty"`
}

// RepoSyncCommandOutput aggregates a batch sync invocation: overall success
// flag, count of repos touched, and the per-repo RepoSyncOutput slice.
//
//	out := RepoSyncCommandOutput{Success: true, Count: 3, Synced: []RepoSyncOutput{...}}
type RepoSyncCommandOutput struct {
	Success bool             `json:"success"`
	Count   int              `json:"count"`
	Synced  []RepoSyncOutput `json:"synced,omitempty"`
}

const (
	repoSyncResetAction    = "sync.reset"
	repoSyncRepoDirContext = "agentic.repoSyncRepoDir"
)

// s.registerRepoSyncSupport()
func (s *PrepSubsystem) registerRepoSyncSupport() {
	if s == nil || s.ServiceRuntime == nil {
		return
	}

	c := s.Core()
	if c == nil {
		return
	}
	if c.Config().Bool("agentic.repo_sync.registered") {
		return
	}

	c.Config().Set("agentic.repo_sync.registered", true)

	if !c.Action("sync.fetch").Exists() {
		c.Action("sync.fetch", s.handleRepoSyncFetch).Description = "Fetch a tracked local repo from origin"
	}
	if !c.Action(repoSyncResetAction).Exists() {
		c.Action(repoSyncResetAction, s.handleRepoSyncReset).Description = "Reset a tracked local repo to origin/<branch>"
	}

	c.RegisterAction(func(coreApp *core.Core, msg core.Message) core.Result {
		return s.handleRepoSyncIPC(coreApp, msg)
	})

	if !c.Command("repo/sync").OK {
		c.Command("repo/sync", core.Command{
			Description: "Fetch and optionally reset tracked local repos from origin",
			Action:      s.cmdRepoSyncLocal,
		})
	}
	if !c.Command("agentic:repo/sync").OK {
		c.Command("agentic:repo/sync", core.Command{
			Description: "Fetch and optionally reset tracked local repos from origin",
			Action:      s.cmdRepoSyncLocal,
		})
	}
}

func (s *PrepSubsystem) handleRepoSyncIPC(_ *core.Core, msg core.Message) core.Result {
	ev, ok := msg.(messages.WorkspacePushed)
	if !ok {
		return core.Result{OK: true}
	}

	result := s.onWorkspacePushed(context.Background(), ev)
	if !result.OK {
		core.Warn(
			"agentic repo sync failed",
			"org", ev.Org,
			"repo", ev.Repo,
			"branch", ev.Branch,
			"reason", result.Value,
		)
	}
	return result
}

// result := s.onWorkspacePushed(ctx, messages.WorkspacePushed{Repo: "go-io", Branch: "main", Org: "core"})
func (s *PrepSubsystem) onWorkspacePushed(ctx context.Context, ev messages.WorkspacePushed) core.Result {
	target, err := repoSyncSingleTarget(ev.Org, ev.Repo)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return s.runRepoSync(ctx, target, ev.Branch, true)
}

func (s *PrepSubsystem) cmdRepoSyncLocal(options core.Options) core.Result {
	targets, err := s.repoSyncTargets(options)
	if err != nil {
		core.Print(nil, "usage: core-agent repo/sync [--repo=go-io] [--org=core] [--branch=main] [--reset]")
		return core.Result{Value: err, OK: false}
	}

	reset := optionBoolValue(options, "reset")
	branch := optionStringValue(options, "branch")
	ctx := s.commandContext()

	output := RepoSyncCommandOutput{
		Success: true,
		Synced:  []RepoSyncOutput{},
	}

	for _, target := range targets {
		result := s.runRepoSync(ctx, target, branch, reset)
		if !result.OK {
			errValue, _ := result.Value.(error)
			if errValue == nil {
				errValue = core.E("agentic.cmdRepoSyncLocal", "repo sync failed", nil)
			}
			core.Print(nil, "error: %v", errValue)
			return core.Result{Value: errValue, OK: false}
		}

		syncOutput, ok := result.Value.(RepoSyncOutput)
		if !ok {
			err := core.E("agentic.cmdRepoSyncLocal", "invalid repo sync output", nil)
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}

		output.Synced = append(output.Synced, syncOutput)
		if syncOutput.Branch != "" {
			core.Print(nil, "fetched %s/%s@%s", syncOutput.Org, syncOutput.Repo, syncOutput.Branch)
		} else {
			core.Print(nil, "fetched %s/%s", syncOutput.Org, syncOutput.Repo)
		}
		if syncOutput.Reset {
			core.Print(nil, "reset %s to origin/%s", syncOutput.RepoDir, syncOutput.Branch)
		}
	}

	output.Count = len(output.Synced)
	core.Print(nil, "count: %d", output.Count)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) runRepoSync(ctx context.Context, target fetchRepoRef, branch string, reset bool) core.Result {
	s.registerRepoSyncSupport()

	repoDir, err := s.repoSyncRepoDir(target)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	fetchBranch := core.Trim(branch)
	resetBranch := ""
	if fetchBranch != "" || reset {
		resetBranch, err = s.repoSyncBranch(repoDir, branch)
		if err != nil {
			return core.Result{Value: err, OK: false}
		}
		if fetchBranch == "" && reset {
			fetchBranch = resetBranch
		}
	}

	fetchResult := s.Core().Action("sync.fetch").Run(ctx, repoSyncOptions(target, fetchBranch))
	if !fetchResult.OK {
		return fetchResult
	}

	if reset {
		resetResult := s.Core().Action(repoSyncResetAction).Run(ctx, repoSyncOptions(target, resetBranch))
		if !resetResult.OK {
			return resetResult
		}
	}

	return core.Result{Value: RepoSyncOutput{
		Success: true,
		Org:     target.Org,
		Repo:    target.Repo,
		Branch:  resetBranch,
		RepoDir: repoDir,
		Reset:   reset,
	}, OK: true}
}

// result := c.Action("sync.fetch").Run(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
func (s *PrepSubsystem) handleRepoSyncFetch(ctx context.Context, options core.Options) core.Result {
	target, err := repoSyncTarget(options)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	repoDir, err := s.repoSyncRepoDir(target)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	args := []string{"git", "fetch", "origin"}
	if branch := core.Trim(optionStringValue(options, "branch")); branch != "" {
		args = append(args, branch)
	}

	result := s.Core().Process().RunIn(repoSyncContext(ctx), repoDir, args...)
	if !result.OK {
		return core.Result{Value: core.E("agentic.handleRepoSyncFetch", "git fetch failed", resultErrorValue(result)), OK: false}
	}

	return core.Result{Value: RepoSyncOutput{
		Success: true,
		Org:     target.Org,
		Repo:    target.Repo,
		Branch:  core.Trim(optionStringValue(options, "branch")),
		RepoDir: repoDir,
	}, OK: true}
}

// result := c.Action("sync.reset").Run(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}, core.Option{Key: "branch", Value: "main"}))
func (s *PrepSubsystem) handleRepoSyncReset(ctx context.Context, options core.Options) core.Result {
	target, err := repoSyncTarget(options)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	repoDir, err := s.repoSyncRepoDir(target)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	branch, err := s.repoSyncBranch(repoDir, optionStringValue(options, "branch"))
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	process := s.Core().Process()
	if current := s.currentBranch(repoDir); current != "" && current != branch {
		checkoutResult := process.RunIn(repoSyncContext(ctx), repoDir, "git", "checkout", "-B", branch, core.Concat("origin/", branch))
		if !checkoutResult.OK {
			return core.Result{Value: core.E("agentic.handleRepoSyncReset", "git checkout failed", resultErrorValue(checkoutResult)), OK: false}
		}
	}

	resetResult := process.RunIn(repoSyncContext(ctx), repoDir, "git", "reset", "--hard", core.Concat("origin/", branch))
	if !resetResult.OK {
		return core.Result{Value: core.E("agentic.handleRepoSyncReset", "git reset failed", resultErrorValue(resetResult)), OK: false}
	}

	return core.Result{Value: RepoSyncOutput{
		Success: true,
		Org:     target.Org,
		Repo:    target.Repo,
		Branch:  branch,
		RepoDir: repoDir,
		Reset:   true,
	}, OK: true}
}

func (s *PrepSubsystem) repoSyncTargets(options core.Options) ([]fetchRepoRef, error) {
	if repo := optionStringValue(options, "repo", "_arg"); repo != "" {
		target, err := repoSyncSingleTarget(optionStringValue(options, "org"), repo)
		if err != nil {
			return nil, err
		}
		return []fetchRepoRef{target}, nil
	}
	return s.fetchLoopRepoRefs(), nil
}

func repoSyncTarget(options core.Options) (fetchRepoRef, error) {
	return repoSyncSingleTarget(optionStringValue(options, "org"), optionStringValue(options, "repo", "_arg"))
}

func repoSyncSingleTarget(orgValue, repoValue string) (fetchRepoRef, error) {
	org := core.Trim(orgValue)
	if org == "" {
		org = "core"
	}

	repoPath := core.Replace(core.Trim(repoValue), "\\", "/")
	if repoPath == "" {
		return fetchRepoRef{}, core.E("agentic.repoSyncSingleTarget", "repo is required", nil)
	}

	if parts := core.Split(repoPath, "/"); len(parts) == 2 {
		if parts[0] != "" {
			org = parts[0]
		}
		repoPath = parts[1]
	}

	validatedOrg, ok := validateName(org)
	if !ok {
		return fetchRepoRef{}, core.E("agentic.repoSyncSingleTarget", "invalid org name", nil)
	}
	validatedRepo, ok := validateName(core.PathBase(repoPath))
	if !ok {
		return fetchRepoRef{}, core.E("agentic.repoSyncSingleTarget", "invalid repo name", nil)
	}

	return fetchRepoRef{Org: validatedOrg, Repo: validatedRepo}, nil
}

func repoSyncOptions(target fetchRepoRef, branch string) core.Options {
	options := core.NewOptions(
		core.Option{Key: "org", Value: target.Org},
		core.Option{Key: "repo", Value: target.Repo},
	)
	if core.Trim(branch) != "" {
		options.Set("branch", core.Trim(branch))
	}
	return options
}

func repoSyncContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func resultErrorValue(result core.Result) error {
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}
	return nil
}

func (s *PrepSubsystem) repoSyncRepoDir(target fetchRepoRef) (string, error) {
	if s == nil || s.ServiceRuntime == nil {
		return "", core.E(repoSyncRepoDirContext, "prep subsystem is not initialised", nil)
	}

	repoDir := s.localRepoDir(target.Org, target.Repo)
	if repoDir == "" || !fs.Exists(repoDir) || fs.IsFile(repoDir) {
		return "", core.E(repoSyncRepoDirContext, "local repo not found", nil)
	}

	if !s.Core().Process().RunIn(context.Background(), repoDir, "git", "rev-parse", "--git-dir").OK {
		return "", core.E(repoSyncRepoDirContext, "local repo is not a git checkout", nil)
	}
	return repoDir, nil
}

func (s *PrepSubsystem) repoSyncBranch(repoDir, branch string) (string, error) {
	targetBranch := core.Trim(branch)
	if targetBranch == "" {
		targetBranch = s.currentBranch(repoDir)
	}
	if targetBranch == "" {
		targetBranch = s.DefaultBranch(repoDir)
	}
	if targetBranch == "" {
		return "", core.E("agentic.repoSyncBranch", "branch is required", nil)
	}
	return targetBranch, nil
}
