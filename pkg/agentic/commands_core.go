// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

// Follow-up: docs/RFC.pipeline.md still does not exist in this checkout.
// This scaffold follows docs/RFC-AGENT-PIPELINE.md section "core pipeline"
// and the repo should restore docs/RFC.pipeline.md as an alias or extracted
// sub-spec so the cross-links stop drifting.

type coreCommandSpec struct {
	Path        string
	Description string
	Usage       string
	NeedsArg    bool
	Next        string
}

var coreCommandSpecs = []coreCommandSpec{
	{
		Path:        "core",
		Description: "Core pipeline command tree",
		Usage:       "core [pipeline] [--help]",
	},
	{
		Path:        "core/pipeline",
		Description: "Run the RFC core pipeline command tree",
		Usage:       "core pipeline [command] [--help]",
	},
	{
		Path:        "core/pipeline/audit",
		Description: "Stage 1: audit issues into implementation work",
		Usage:       "core pipeline audit <repo> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-audit-issues.md and wire a concrete audit handler into this scaffold.",
	},
	{
		Path:        "core/pipeline/epic",
		Description: "Epic pipeline commands",
		Usage:       "core pipeline epic [create|run|status|sync] [--help]",
	},
	{
		Path:        "core/pipeline/epic/create",
		Description: "Stage 2: group implementation issues into epics",
		Usage:       "core pipeline epic create <repo> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-create-epic.md and route this command to an epic creation handler.",
	},
	{
		Path:        "core/pipeline/epic/run",
		Description: "Stage 3: dispatch and monitor an epic",
		Usage:       "core pipeline epic run <epic-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-issue-epic.md and add an epic execution handler for this command.",
	},
	{
		Path:        "core/pipeline/epic/status",
		Description: "Show epic progress",
		Usage:       "core pipeline epic status [epic-number] [--help]",
		Next:        "Read docs/flow/RFC.flow-issue-epic.md and add a status reader for epic progress.",
	},
	{
		Path:        "core/pipeline/epic/sync",
		Description: "Sync epic checklist state from child issues",
		Usage:       "core pipeline epic sync <epic-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-issue-epic.md and add checklist sync against child issue state.",
	},
	{
		Path:        "core/pipeline/monitor",
		Description: "Watch open PRs and intervene when the pipeline stalls",
		Usage:       "core pipeline monitor [repo] [--help]",
		Next:        "Read docs/RFC-AGENT-PIPELINE.md and route this command to a real PR monitor.",
	},
	{
		Path:        "core/pipeline/fix",
		Description: "Pipeline fix-up commands",
		Usage:       "core pipeline fix [reviews|conflicts|format|threads] [--help]",
	},
	{
		Path:        "core/pipeline/fix/reviews",
		Description: "Fix code review findings for a pull request",
		Usage:       "core pipeline fix reviews <pr-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-issue-epic.md and add review-fix orchestration for the PR.",
	},
	{
		Path:        "core/pipeline/fix/conflicts",
		Description: "Fix merge conflicts for a pull request",
		Usage:       "core pipeline fix conflicts <pr-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-resolve-stuck-prs.md and add conflict resolution for the PR.",
	},
	{
		Path:        "core/pipeline/fix/format",
		Description: "Apply formatting-only fixes for a pull request",
		Usage:       "core pipeline fix format <pr-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-issue-epic.md and add a formatting-only fast path.",
	},
	{
		Path:        "core/pipeline/fix/threads",
		Description: "Resolve review threads after a fix lands",
		Usage:       "core pipeline fix threads <pr-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-issue-epic.md and add thread resolution against PR metadata.",
	},
	{
		Path:        "core/pipeline/onboard",
		Description: "Run the full audit -> epic -> dispatch onboarding flow",
		Usage:       "core pipeline onboard <repo> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-issue-orchestrator.md and route this command to the end-to-end onboarding flow.",
	},
	{
		Path:        "core/pipeline/budget",
		Description: "Pipeline budget commands",
		Usage:       "core pipeline budget [plan|log] [--help]",
	},
	{
		Path:        "core/pipeline/budget/plan",
		Description: "Show the optimal dispatch plan for the current budget",
		Usage:       "core pipeline budget plan [--help]",
		Next:        "Read docs/RFC-AGENT-PIPELINE.md and add budget planning based on dispatch constraints.",
	},
	{
		Path:        "core/pipeline/budget/log",
		Description: "Append a dispatch event to the budget journal",
		Usage:       "core pipeline budget log [--help]",
		Next:        "Read docs/RFC-AGENT-PIPELINE.md and add budget event journalling for dispatches.",
	},
	{
		Path:        "core/pipeline/training",
		Description: "Training data capture commands",
		Usage:       "core pipeline training [capture|stats|export] [--help]",
	},
	{
		Path:        "core/pipeline/training/capture",
		Description: "Capture a merged pull request for training data",
		Usage:       "core pipeline training capture <pr-number> [--help]",
		NeedsArg:    true,
		Next:        "Read docs/flow/RFC.flow-gather-training-data.md and add merged-PR capture into the journal.",
	},
	{
		Path:        "core/pipeline/training/stats",
		Description: "Summarise captured training journal data",
		Usage:       "core pipeline training stats [--help]",
		Next:        "Read docs/flow/RFC.flow-gather-training-data.md and add training journal summaries.",
	},
	{
		Path:        "core/pipeline/training/export",
		Description: "Export captured training data for LEM ingestion",
		Usage:       "core pipeline training export [--help]",
		Next:        "Read docs/flow/RFC.flow-gather-training-data.md and add a clean export path for LEM.",
	},
}

func (s *PrepSubsystem) registerCoreCommands() {
	c := s.Core()
	actions := map[string]core.CommandAction{
		"core":                           s.cmdCore,
		"core/pipeline":                  s.cmdCorePipeline,
		"core/pipeline/audit":            s.cmdCorePipelineAudit,
		"core/pipeline/epic":             s.cmdCorePipelineEpic,
		"core/pipeline/epic/create":      s.cmdCorePipelineEpicCreate,
		"core/pipeline/epic/run":         s.cmdCorePipelineEpicRun,
		"core/pipeline/epic/status":      s.cmdCorePipelineEpicStatus,
		"core/pipeline/epic/sync":        s.cmdCorePipelineEpicSync,
		"core/pipeline/monitor":          s.cmdCorePipelineMonitor,
		"core/pipeline/fix":              s.cmdCorePipelineFix,
		"core/pipeline/fix/reviews":      s.cmdCorePipelineFixReviews,
		"core/pipeline/fix/conflicts":    s.cmdCorePipelineFixConflicts,
		"core/pipeline/fix/format":       s.cmdCorePipelineFixFormat,
		"core/pipeline/fix/threads":      s.cmdCorePipelineFixThreads,
		"core/pipeline/onboard":          s.cmdCorePipelineOnboard,
		"core/pipeline/budget":           s.cmdCorePipelineBudget,
		"core/pipeline/budget/plan":      s.cmdCorePipelineBudgetPlan,
		"core/pipeline/budget/log":       s.cmdCorePipelineBudgetLog,
		"core/pipeline/training":         s.cmdCorePipelineTraining,
		"core/pipeline/training/capture": s.cmdCorePipelineTrainingCapture,
		"core/pipeline/training/stats":   s.cmdCorePipelineTrainingStats,
		"core/pipeline/training/export":  s.cmdCorePipelineTrainingExport,
	}

	for _, spec := range coreCommandSpecs {
		c.Command(spec.Path, core.Command{
			Description: spec.Description,
			Action:      actions[spec.Path],
		})
	}
}

func (s *PrepSubsystem) cmdCore(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	if action == "" || action == "help" || optionBoolValue(options, "help") {
		printCoreCommandHelp("core")
		return core.Result{OK: true}
	}

	printCoreCommandHelp("core")
	return core.Result{
		Value: core.E("agentic.cmdCore", core.Concat("unknown core command: ", action), nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdCorePipeline(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	if action == "" || action == "help" || optionBoolValue(options, "help") {
		printCoreCommandHelp("core/pipeline")
		return core.Result{OK: true}
	}

	printCoreCommandHelp("core/pipeline")
	return core.Result{
		Value: core.E("agentic.cmdCorePipeline", core.Concat("unknown core pipeline command: ", action), nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdCorePipelineEpic(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	if action == "" || action == "help" || optionBoolValue(options, "help") {
		printCoreCommandHelp("core/pipeline/epic")
		return core.Result{OK: true}
	}

	printCoreCommandHelp("core/pipeline/epic")
	return core.Result{
		Value: core.E("agentic.cmdCorePipelineEpic", core.Concat("unknown core pipeline epic command: ", action), nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdCorePipelineFix(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	if action == "" || action == "help" || optionBoolValue(options, "help") {
		printCoreCommandHelp("core/pipeline/fix")
		return core.Result{OK: true}
	}

	printCoreCommandHelp("core/pipeline/fix")
	return core.Result{
		Value: core.E("agentic.cmdCorePipelineFix", core.Concat("unknown core pipeline fix command: ", action), nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdCorePipelineBudget(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	if action == "" || action == "help" || optionBoolValue(options, "help") {
		printCoreCommandHelp("core/pipeline/budget")
		return core.Result{OK: true}
	}

	printCoreCommandHelp("core/pipeline/budget")
	return core.Result{
		Value: core.E("agentic.cmdCorePipelineBudget", core.Concat("unknown core pipeline budget command: ", action), nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdCorePipelineTraining(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	if action == "" || action == "help" || optionBoolValue(options, "help") {
		printCoreCommandHelp("core/pipeline/training")
		return core.Result{OK: true}
	}

	printCoreCommandHelp("core/pipeline/training")
	return core.Result{
		Value: core.E("agentic.cmdCorePipelineTraining", core.Concat("unknown core pipeline training command: ", action), nil),
		OK:    false,
	}
}

func (s *PrepSubsystem) cmdCorePipelineAudit(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/audit")
}

func (s *PrepSubsystem) cmdCorePipelineEpicCreate(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/epic/create")
}

func (s *PrepSubsystem) cmdCorePipelineEpicRun(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/epic/run")
}

func (s *PrepSubsystem) cmdCorePipelineEpicStatus(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/epic/status")
}

func (s *PrepSubsystem) cmdCorePipelineEpicSync(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/epic/sync")
}

func (s *PrepSubsystem) cmdCorePipelineMonitor(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/monitor")
}

func (s *PrepSubsystem) cmdCorePipelineFixReviews(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/fix/reviews")
}

func (s *PrepSubsystem) cmdCorePipelineFixConflicts(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/fix/conflicts")
}

func (s *PrepSubsystem) cmdCorePipelineFixFormat(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/fix/format")
}

func (s *PrepSubsystem) cmdCorePipelineFixThreads(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/fix/threads")
}

func (s *PrepSubsystem) cmdCorePipelineOnboard(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/onboard")
}

func (s *PrepSubsystem) cmdCorePipelineBudgetPlan(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/budget/plan")
}

func (s *PrepSubsystem) cmdCorePipelineBudgetLog(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/budget/log")
}

func (s *PrepSubsystem) cmdCorePipelineTrainingCapture(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/training/capture")
}

func (s *PrepSubsystem) cmdCorePipelineTrainingStats(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/training/stats")
}

func (s *PrepSubsystem) cmdCorePipelineTrainingExport(options core.Options) core.Result {
	return runCoreCommandPlaceholder(options, "core/pipeline/training/export")
}

func runCoreCommandPlaceholder(options core.Options, path string) core.Result {
	spec, ok := coreCommandSpecFor(path)
	if !ok {
		return core.Result{
			Value: core.E("agentic.runCoreCommandPlaceholder", core.Concat("unknown core command spec: ", path), nil),
			OK:    false,
		}
	}

	if optionBoolValue(options, "help") {
		printCoreCommandLeafHelp(spec)
		return core.Result{OK: true}
	}

	if spec.NeedsArg && optionStringValue(options, "_arg") == "" {
		printCoreCommandLeafHelp(spec)
		return core.Result{
			Value: core.E(coreCommandErrorName(path), core.Concat("missing required argument for ", coreCommandDisplayPath(path)), nil),
			OK:    false,
		}
	}

	printCoreCommandLeafHelp(spec)
	core.Print(nil, "status: not yet implemented")
	core.Print(nil, "next:   %s", spec.Next)
	return core.Result{
		Value: core.E(coreCommandErrorName(path), core.Concat(coreCommandDisplayPath(path), " is not yet implemented"), nil),
		OK:    false,
	}
}

func printCoreCommandHelp(path string) {
	spec, ok := coreCommandSpecFor(path)
	if !ok {
		return
	}

	core.Print(nil, "usage: %s", spec.Usage)
	core.Print(nil, "")
	core.Print(nil, "subcommands:")

	prefix := core.Concat(path, "/")
	count := 0
	for _, child := range coreCommandSpecs {
		if child.Path == path || !core.HasPrefix(child.Path, prefix) {
			continue
		}
		core.Print(nil, "  %-44s %s", child.Usage, child.Description)
		count++
	}

	if count == 0 {
		core.Print(nil, "  none")
	}
}

func printCoreCommandLeafHelp(spec coreCommandSpec) {
	core.Print(nil, "usage: %s", spec.Usage)
	core.Print(nil, "about: %s", spec.Description)
}

func coreCommandSpecFor(path string) (coreCommandSpec, bool) {
	for _, spec := range coreCommandSpecs {
		if spec.Path == path {
			return spec, true
		}
	}
	return coreCommandSpec{}, false
}

func coreCommandDisplayPath(path string) string {
	return core.Replace(path, "/", " ")
}

func coreCommandErrorName(path string) string {
	return core.Concat("agentic.", core.Replace(path, "/", "."))
}
