// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

type PipelineOnboardInput struct {
	Org            string `json:"org,omitempty"`
	Repo           string `json:"repo"`
	Agent          string `json:"agent,omitempty"`
	Template       string `json:"template,omitempty"`
	DispatchDryRun bool   `json:"dispatch_dry_run,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type PipelineOnboardOutput struct {
	Success bool                     `json:"success"`
	Org     string                   `json:"org,omitempty"`
	Repo    string                   `json:"repo"`
	Audit   PipelineAuditOutput      `json:"audit"`
	Epic    PipelineEpicCreateOutput `json:"epic"`
	Runs    []PipelineEpicRunOutput  `json:"runs,omitempty"`
	Direct  []PipelineIssueRef       `json:"direct,omitempty"`
}

func (s *PrepSubsystem) cmdPipelineOnboard(options core.Options) core.Result {
	ctx := s.commandContext()
	repo := pipelineRepoValue(options)
	if repo == "" {
		core.Print(nil, "usage: core-agent pipeline/onboard <repo> [--org=core] [--agent=codex] [--template=coding] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineOnboard", "repo is required", nil), OK: false}
	}

	output, err := pipelineOnboard(s, ctx, PipelineOnboardInput{
		Org:            pipelineOrgValue(options),
		Repo:           repo,
		Agent:          optionStringValue(options, "agent"),
		Template:       optionStringValue(options, "template"),
		DispatchDryRun: optionBoolValue(options, "dry_run", "dry-run"),
		Limit:          optionIntValue(options, "limit"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "repo:          %s/%s", output.Org, output.Repo)
	core.Print(nil, "audit created: %d", len(output.Audit.Created))
	core.Print(nil, "epics:         %d", len(output.Epic.Epics))
	core.Print(nil, "epic runs:     %d", len(output.Runs))
	core.Print(nil, "direct:        %d", len(output.Direct))
	for _, run := range output.Runs {
		core.Print(nil, "  epic #%d dispatched %d issue(s)", run.EpicNumber, len(run.Dispatched))
	}
	return core.Result{Value: output, OK: true}
}

var pipelineOnboard = func(s *PrepSubsystem, ctx context.Context, input PipelineOnboardInput) (PipelineOnboardOutput, error) {
	if input.Repo == "" {
		return PipelineOnboardOutput{}, core.E("pipelineOnboard", "repo is required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Agent == "" {
		input.Agent = "codex"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	auditOutput, err := pipelineAudit(s, ctx, PipelineAuditInput{
		Org:  input.Org,
		Repo: input.Repo,
	})
	if err != nil {
		return PipelineOnboardOutput{}, err
	}

	epicOutput, err := pipelineEpicCreate(s, ctx, PipelineEpicCreateInput{
		Org:  input.Org,
		Repo: input.Repo,
	})
	if err != nil {
		return PipelineOnboardOutput{}, err
	}

	output := PipelineOnboardOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
		Audit:   auditOutput,
		Epic:    epicOutput,
	}

	epicsToRun := make([]PipelineEpicMeta, 0, len(epicOutput.Epics)+len(epicOutput.Existing))
	epicsToRun = append(epicsToRun, epicOutput.Epics...)
	epicsToRun = append(epicsToRun, epicOutput.Existing...)

	if len(epicsToRun) == 0 {
		direct, directErr := pipelineOnboardDispatchDirect(s, ctx, input, epicOutput.Candidates)
		if directErr != nil {
			return PipelineOnboardOutput{}, directErr
		}
		output.Direct = direct
		return output, nil
	}

	for _, epic := range epicsToRun {
		runOutput, runErr := pipelineEpicRun(s, ctx, PipelineEpicRunInput{
			Org:        input.Org,
			Repo:       input.Repo,
			EpicNumber: epic.Number,
			Agent:      input.Agent,
			Template:   input.Template,
			DryRun:     input.DispatchDryRun,
			Limit:      input.Limit,
			Epic:       &epic,
		})
		if runErr != nil {
			return PipelineOnboardOutput{}, runErr
		}
		output.Runs = append(output.Runs, runOutput)
	}

	return output, nil
}

var pipelineOnboardDispatchDirect = func(s *PrepSubsystem, ctx context.Context, input PipelineOnboardInput, issues []PipelineIssueRef) ([]PipelineIssueRef, error) {
	dispatched := []PipelineIssueRef{}
	for _, issue := range issues {
		if issue.Number <= 0 || !pipelineIssueAutoDispatchAllowed(issue.Title) {
			continue
		}
		if input.Limit > 0 && len(dispatched) >= input.Limit {
			break
		}

		if input.DispatchDryRun {
			dispatched = append(dispatched, issue)
			continue
		}

		_, _, err := dispatch(s, ctx, nil, DispatchInput{
			Org:      input.Org,
			Repo:     input.Repo,
			Task:     issue.Title,
			Issue:    issue.Number,
			Agent:    input.Agent,
			Template: input.Template,
		})
		if err != nil {
			return nil, core.E("pipelineOnboard", core.Concat("failed to dispatch issue #", core.Sprint(issue.Number)), err)
		}
		dispatched = append(dispatched, issue)
	}
	return dispatched, nil
}
