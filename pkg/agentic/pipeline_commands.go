// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"regexp"
	"unicode"

	core "dappco.re/go/core"
)

var pipelineNumberPattern = regexp.MustCompile(`^[0-9]+$`)

func (s *PrepSubsystem) registerPipelineCommands() {
	c := s.Core()

	c.Command("pipeline", core.Command{Description: "Run the agent pipeline command tree", Action: s.cmdPipeline})
	c.Command("agentic:pipeline", core.Command{Description: "Run the agent pipeline command tree", Action: s.cmdPipeline})
	c.Command("pipeline/audit", core.Command{Description: "Stage 1: audit issues into implementation work", Action: s.cmdPipelineAudit})
	c.Command("agentic:pipeline/audit", core.Command{Description: "Stage 1: audit issues into implementation work", Action: s.cmdPipelineAudit})
	c.Command("pipeline/epic", core.Command{Description: "Stage 2 and 3 epic orchestration commands", Action: s.cmdPipelineEpic})
	c.Command("agentic:pipeline/epic", core.Command{Description: "Stage 2 and 3 epic orchestration commands", Action: s.cmdPipelineEpic})
	c.Command("pipeline/epic/create", core.Command{Description: "Group implementation issues into epics", Action: s.cmdPipelineEpicCreate})
	c.Command("agentic:pipeline/epic/create", core.Command{Description: "Group implementation issues into epics", Action: s.cmdPipelineEpicCreate})
	c.Command("pipeline/epic/run", core.Command{Description: "Dispatch and monitor an epic", Action: s.cmdPipelineEpicRun})
	c.Command("agentic:pipeline/epic/run", core.Command{Description: "Dispatch and monitor an epic", Action: s.cmdPipelineEpicRun})
	c.Command("pipeline/epic/status", core.Command{Description: "Show epic progress", Action: s.cmdPipelineEpicStatus})
	c.Command("agentic:pipeline/epic/status", core.Command{Description: "Show epic progress", Action: s.cmdPipelineEpicStatus})
	c.Command("pipeline/epic/sync", core.Command{Description: "Sync epic checklist state from child issues", Action: s.cmdPipelineEpicSync})
	c.Command("agentic:pipeline/epic/sync", core.Command{Description: "Sync epic checklist state from child issues", Action: s.cmdPipelineEpicSync})
	c.Command("pipeline/monitor", core.Command{Description: "Watch open PRs and auto-intervene", Action: s.cmdPipelineMonitor})
	c.Command("agentic:pipeline/monitor", core.Command{Description: "Watch open PRs and auto-intervene", Action: s.cmdPipelineMonitor})
	c.Command("pipeline/fix", core.Command{Description: "Pipeline fix-up commands", Action: s.cmdPipelineFix})
	c.Command("agentic:pipeline/fix", core.Command{Description: "Pipeline fix-up commands", Action: s.cmdPipelineFix})
	c.Command("pipeline/fix/reviews", core.Command{Description: "Ask the agent to fix code reviews on a pull request", Action: s.cmdPipelineFixReviews})
	c.Command("agentic:pipeline/fix/reviews", core.Command{Description: "Ask the agent to fix code reviews on a pull request", Action: s.cmdPipelineFixReviews})
	c.Command("pipeline/fix/conflicts", core.Command{Description: "Ask the agent to fix a merge conflict on a pull request", Action: s.cmdPipelineFixConflicts})
	c.Command("agentic:pipeline/fix/conflicts", core.Command{Description: "Ask the agent to fix a merge conflict on a pull request", Action: s.cmdPipelineFixConflicts})
	c.Command("pipeline/fix/format", core.Command{Description: "Apply formatting-only fixes in a workspace or repo checkout", Action: s.cmdPipelineFixFormat})
	c.Command("agentic:pipeline/fix/format", core.Command{Description: "Apply formatting-only fixes in a workspace or repo checkout", Action: s.cmdPipelineFixFormat})
	c.Command("pipeline/fix/threads", core.Command{Description: "Handle review-thread follow-up for a pull request", Action: s.cmdPipelineFixThreads})
	c.Command("agentic:pipeline/fix/threads", core.Command{Description: "Handle review-thread follow-up for a pull request", Action: s.cmdPipelineFixThreads})
	c.Command("pipeline/onboard", core.Command{Description: "Run audit, epic creation, and dispatch onboarding for a repo", Action: s.cmdPipelineOnboard})
	c.Command("agentic:pipeline/onboard", core.Command{Description: "Run audit, epic creation, and dispatch onboarding for a repo", Action: s.cmdPipelineOnboard})
	c.Command("pipeline/budget", core.Command{Description: "Budget planning commands", Action: s.cmdPipelineBudget})
	c.Command("agentic:pipeline/budget", core.Command{Description: "Budget planning commands", Action: s.cmdPipelineBudget})
	c.Command("pipeline/budget/plan", core.Command{Description: "Show daily dispatch budget planning", Action: s.cmdPipelineBudgetPlan})
	c.Command("agentic:pipeline/budget/plan", core.Command{Description: "Show daily dispatch budget planning", Action: s.cmdPipelineBudgetPlan})
	c.Command("pipeline/budget/log", core.Command{Description: "Append a dispatch event to the budget journal", Action: s.cmdPipelineBudgetLog})
	c.Command("agentic:pipeline/budget/log", core.Command{Description: "Append a dispatch event to the budget journal", Action: s.cmdPipelineBudgetLog})
	c.Command("pipeline/training", core.Command{Description: "Training journal commands", Action: s.cmdPipelineTraining})
	c.Command("agentic:pipeline/training", core.Command{Description: "Training journal commands", Action: s.cmdPipelineTraining})
	c.Command("pipeline/training/capture", core.Command{Description: "Capture a merged pull request for training", Action: s.cmdPipelineTrainingCapture})
	c.Command("agentic:pipeline/training/capture", core.Command{Description: "Capture a merged pull request for training", Action: s.cmdPipelineTrainingCapture})
	c.Command("pipeline/training/stats", core.Command{Description: "Summarise training journal data", Action: s.cmdPipelineTrainingStats})
	c.Command("agentic:pipeline/training/stats", core.Command{Description: "Summarise training journal data", Action: s.cmdPipelineTrainingStats})
	c.Command("pipeline/training/export", core.Command{Description: "Export training journal data", Action: s.cmdPipelineTrainingExport})
	c.Command("agentic:pipeline/training/export", core.Command{Description: "Export training journal data", Action: s.cmdPipelineTrainingExport})
}

func (s *PrepSubsystem) cmdPipeline(options core.Options) core.Result {
	switch action := optionStringValue(options, "action", "_arg"); action {
	case "", "help":
		printPipelineUsage()
		return core.Result{OK: true}
	case "audit":
		return s.cmdPipelineAudit(options)
	case "epic":
		return s.cmdPipelineEpic(options)
	case "monitor":
		return s.cmdPipelineMonitor(options)
	case "fix":
		return s.cmdPipelineFix(options)
	case "onboard":
		return s.cmdPipelineOnboard(options)
	case "budget":
		return s.cmdPipelineBudget(options)
	case "training":
		return s.cmdPipelineTraining(options)
	default:
		printPipelineUsage()
		return core.Result{Value: core.E("agentic.cmdPipeline", core.Concat("unknown pipeline command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdPipelineEpic(options core.Options) core.Result {
	switch action := optionStringValue(options, "action", "_arg"); action {
	case "", "help":
		printPipelineEpicUsage()
		return core.Result{OK: true}
	case "create":
		return s.cmdPipelineEpicCreate(options)
	case "run":
		return s.cmdPipelineEpicRun(options)
	case "status":
		return s.cmdPipelineEpicStatus(options)
	case "sync":
		return s.cmdPipelineEpicSync(options)
	default:
		printPipelineEpicUsage()
		return core.Result{Value: core.E("agentic.cmdPipelineEpic", core.Concat("unknown pipeline epic command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdPipelineFix(options core.Options) core.Result {
	switch action := optionStringValue(options, "action", "_arg"); action {
	case "", "help":
		printPipelineFixUsage()
		return core.Result{OK: true}
	case "reviews":
		return s.cmdPipelineFixReviews(options)
	case "conflicts":
		return s.cmdPipelineFixConflicts(options)
	case "format":
		return s.cmdPipelineFixFormat(options)
	case "threads":
		return s.cmdPipelineFixThreads(options)
	default:
		printPipelineFixUsage()
		return core.Result{Value: core.E("agentic.cmdPipelineFix", core.Concat("unknown pipeline fix command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdPipelineBudget(options core.Options) core.Result {
	switch action := optionStringValue(options, "action", "_arg"); action {
	case "", "help":
		printPipelineBudgetUsage()
		return core.Result{OK: true}
	case "plan":
		return s.cmdPipelineBudgetPlan(options)
	case "log":
		return s.cmdPipelineBudgetLog(options)
	default:
		printPipelineBudgetUsage()
		return core.Result{Value: core.E("agentic.cmdPipelineBudget", core.Concat("unknown pipeline budget command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdPipelineTraining(options core.Options) core.Result {
	switch action := optionStringValue(options, "action", "_arg"); action {
	case "", "help":
		printPipelineTrainingUsage()
		return core.Result{OK: true}
	case "capture":
		return s.cmdPipelineTrainingCapture(options)
	case "stats":
		return s.cmdPipelineTrainingStats(options)
	case "export":
		return s.cmdPipelineTrainingExport(options)
	default:
		printPipelineTrainingUsage()
		return core.Result{Value: core.E("agentic.cmdPipelineTraining", core.Concat("unknown pipeline training command: ", action), nil), OK: false}
	}
}

func pipelineOrgValue(options core.Options) string {
	org := optionStringValue(options, "org")
	if org == "" {
		org = "core"
	}
	return org
}

func pipelineRepoValue(options core.Options) string {
	repo := optionStringValue(options, "repo")
	if repo != "" {
		return repo
	}

	arg := optionStringValue(options, "_arg")
	if arg == "" || pipelineNumberPattern.MatchString(core.Trim(arg)) {
		return ""
	}

	return arg
}

func pipelineRepoAndNumberValue(options core.Options) (string, string, int) {
	org := pipelineOrgValue(options)
	repo := pipelineRepoValue(options)
	number := optionIntValue(options, "number")
	if number == 0 {
		arg := optionStringValue(options, "_arg")
		if pipelineNumberPattern.MatchString(core.Trim(arg)) {
			number = parseIntString(arg)
		}
	}
	return org, repo, number
}

func pipelineWorkspaceDir(options core.Options) string {
	workspace := optionStringValue(options, "workspace")
	if workspace != "" {
		return core.JoinPath(WorkspaceRoot(), workspace, "repo")
	}
	return optionStringValue(options, "repo_dir", "repo-dir", "path")
}

func pipelineSlug(value string) string {
	normalised := core.Lower(core.Trim(value))
	if normalised == "" {
		return "pipeline"
	}

	lastDash := false
	out := make([]rune, 0, len(normalised))
	for _, r := range normalised {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		out = append(out, '-')
		lastDash = true
	}

	slug := string(out)
	for len(slug) > 0 && slug[0] == '-' {
		slug = slug[1:]
	}
	for len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}
	if slug == "" {
		return "pipeline"
	}
	return slug
}

func pipelineDisplayTheme(theme string) string {
	switch theme {
	case "security":
		return "Security"
	case "testing":
		return "Testing"
	case "docs":
		return "Docs"
	case "performance":
		return "Performance"
	case "features":
		return "Features"
	default:
		return "Quality"
	}
}

func printPipelineUsage() {
	core.Print(nil, "usage: core-agent pipeline [audit|epic|monitor|fix|onboard|budget|training]")
	core.Print(nil, "       core-agent pipeline/audit <repo> [--org=core] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/epic/create <repo> [--org=core] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/epic/run <epic-number> --repo=<repo> [--agent=codex] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/monitor [<repo>] [--org=core] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/fix/reviews <pr-number> --repo=<repo> [--org=core] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/onboard <repo> [--org=core] [--agent=codex] [--dry-run]")
}

func printPipelineEpicUsage() {
	core.Print(nil, "usage: core-agent pipeline/epic/create <repo> [--org=core] [--theme=security] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/epic/run <epic-number> --repo=<repo> [--org=core] [--agent=codex] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/epic/status <epic-number> --repo=<repo> [--org=core]")
	core.Print(nil, "       core-agent pipeline/epic/sync <epic-number> --repo=<repo> [--org=core] [--dry-run]")
}

func printPipelineFixUsage() {
	core.Print(nil, "usage: core-agent pipeline/fix/reviews <pr-number> --repo=<repo> [--org=core] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/fix/conflicts <pr-number> --repo=<repo> [--org=core] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/fix/format <pr-number> --repo=<repo> [--org=core] [--workspace=<name>|--repo-dir=<path>] [--commit] [--push] [--dry-run]")
	core.Print(nil, "       core-agent pipeline/fix/threads <pr-number> --repo=<repo> [--org=core] [--dry-run]")
}

func printPipelineBudgetUsage() {
	core.Print(nil, "usage: core-agent pipeline/budget/plan")
	core.Print(nil, "       core-agent pipeline/budget/log")
}

func printPipelineTrainingUsage() {
	core.Print(nil, "usage: core-agent pipeline/training/capture <pr-number> --repo=<repo> [--org=core]")
	core.Print(nil, "       core-agent pipeline/training/stats")
	core.Print(nil, "       core-agent pipeline/training/export")
}
