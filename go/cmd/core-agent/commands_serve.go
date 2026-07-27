// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lemma"
)

// CLI surface for reading/controlling the local lthn-mlx serve via
// /v1/admin/*. Bearer auth loads from ~/Lethean/data/admin.token by
// default; override with --admin-token=<value> or
// --admin-token-file=<path>.
//
//	core-agent serve-status
//	core-agent serve-reload --confirm=<machine-hash> --model=/Lethean/models/lemer-lite
//	core-agent serve-profiles
//	core-agent serve-status --base-url=http://192.168.1.50:11434
//
// The "serve-" prefix mirrors lthn-mlx's "serve" subcommand — both
// halves of the conversation use the same word. We hyphen-prefix
// rather than space-separate because the core.Command API is flat
// (no native sub-verb support).

// serveStatus prints the boot-time snapshot the engine was started
// with (post-profile, post-context-override). Useful for "what's
// actually loaded?" without grepping the engine's stderr.
func (commands applicationCommandSet) serveStatus(opts core.Options) core.Result {
	admin, ok := buildAdmin(opts)
	if !ok {
		return core.Result{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	st, err := admin.Status(ctx)
	if err != nil {
		applicationPrint("serve-status: %v", err)
		return core.Result{}
	}
	applicationPrint("serve-status")
	applicationPrint("  model:        %s", st.ModelPath)
	if st.ProfilePath != "" {
		applicationPrint("  profile:      %s", st.ProfilePath)
	}
	applicationPrint("  runtime:      %s", st.Runtime)
	applicationPrint("  loaded:       %s", core.TimeFormat(time.Unix(st.LoadedAtUnix, 0), time.RFC3339))
	applicationPrint("  context:      %d", st.Config.ContextLength)
	applicationPrint("  slots:        %d", st.Config.ParallelSlots)
	applicationPrint("  cache:        prompt=%v policy=%s mode=%s",
		st.Config.PromptCache, st.Config.CachePolicy, st.Config.CacheMode)
	if st.Config.BatchSize > 0 {
		applicationPrint("  batch:        %d (prefill chunk %d)", st.Config.BatchSize, st.Config.PrefillChunkSize)
	}
	if st.Config.AdapterPath != "" {
		applicationPrint("  adapter:      %s", st.Config.AdapterPath)
	}
	return core.Result{OK: true}
}

// serveReload hot-swaps the loaded model without restarting the
// process. --confirm must match the running machine hash (read via
// `core-agent serve-status` first); the gate stops accidental
// reload of the wrong instance when one operator manages several.
func (commands applicationCommandSet) serveReload(opts core.Options) core.Result {
	confirm := opts.String("confirm")
	model := opts.String("model")
	profile := opts.String("profile")
	ctxLen := opts.Int("context")
	if confirm == "" {
		applicationPrint("serve-reload: --confirm=<machine-hash> required (use `serve-status` to read)")
		return core.Result{}
	}
	if model == "" && profile == "" && ctxLen == 0 {
		applicationPrint("serve-reload: nothing to do — pass --model, --profile, and/or --context")
		return core.Result{}
	}
	admin, ok := buildAdmin(opts)
	if !ok {
		return core.Result{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err := admin.Reload(ctx, lemma.ReloadRequest{
		ConfirmMachine: confirm,
		ModelPath:      model,
		ProfilePath:    profile,
		ContextLength:  ctxLen,
	})
	if err != nil {
		applicationPrint("serve-reload: %v", err)
		return core.Result{}
	}
	applicationPrint("serve-reload: ok")
	return core.Result{OK: true}
}

// serveProfiles lists tuning profiles the engine sees in its standard
// directory. Names map 1:1 to the --profile argument of serve-reload.
func (commands applicationCommandSet) serveProfiles(opts core.Options) core.Result {
	admin, ok := buildAdmin(opts)
	if !ok {
		return core.Result{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pl, err := admin.Profiles(ctx)
	if err != nil {
		applicationPrint("serve-profiles: %v", err)
		return core.Result{}
	}
	applicationPrint("profiles in %s", pl.Dir)
	if len(pl.Profiles) == 0 {
		applicationPrint("  (none)")
		return core.Result{OK: true}
	}
	for _, p := range pl.Profiles {
		applicationPrint("  %s  (backend=%s model=%s)", p.Name, p.Backend, p.Model)
	}
	return core.Result{OK: true}
}

// buildAdmin resolves a lemma.Admin from CLI options. Returns ok=false
// + prints the user-visible reason when config fails. Pattern reused
// by both serve-* and models-* commands.
func buildAdmin(opts core.Options) (*lemma.Admin, bool) {
	cfg := lemma.AdminConfig{
		BaseURL:   opts.String("base-url"),
		Token:     opts.String("admin-token"),
		TokenPath: opts.String("admin-token-file"),
	}
	admin, err := lemma.NewAdmin(cfg)
	if err != nil {
		applicationPrint("admin client: %v", err)
		applicationPrint("  hint: lthn-mlx writes the token to ~/Lethean/data/admin.token on first boot")
		applicationPrint("        pass --admin-token=<value> or --admin-token-file=<path> to override")
		return nil, false
	}
	return admin, true
}
