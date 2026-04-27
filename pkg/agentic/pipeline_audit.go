// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"regexp"

	core "dappco.re/go/core"
)

var pipelineBulletPattern = regexp.MustCompile(`^\s*(?:[-*]|\d+\.)\s+(.*)$`)
var pipelineWhitespacePattern = regexp.MustCompile(`\s+`)

type PipelineIssueRef struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state,omitempty"`
	URL    string   `json:"url,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

type PipelineAuditInput struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo"`
	DryRun bool   `json:"dry_run,omitempty"`
}

type PipelineAuditOutput struct {
	Success  bool               `json:"success"`
	Org      string             `json:"org,omitempty"`
	Repo     string             `json:"repo"`
	Audits   []PipelineIssueRef `json:"audits"`
	Created  []PipelineIssueRef `json:"created"`
	Existing []PipelineIssueRef `json:"existing"`
	Closed   []int              `json:"closed,omitempty"`
}

type pipelineRepoRecord struct {
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
}

type pipelineLabelRecord struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type pipelineIssueRecord struct {
	Number      int                   `json:"number"`
	Title       string                `json:"title"`
	Body        string                `json:"body"`
	State       string                `json:"state"`
	HTMLURL     string                `json:"html_url"`
	Labels      []pipelineLabelRecord `json:"labels"`
	PullRequest map[string]any        `json:"pull_request"`
}

func (s *PrepSubsystem) cmdPipelineAudit(options core.Options) core.Result {
	ctx := s.commandContext()
	org := pipelineOrgValue(options)
	repo := pipelineRepoValue(options)
	if repo == "" {
		core.Print(nil, "usage: core-agent pipeline/audit <repo> [--org=core] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineAudit", "repo is required", nil), OK: false}
	}

	output, err := s.pipelineAudit(ctx, PipelineAuditInput{
		Org:    org,
		Repo:   repo,
		DryRun: optionBoolValue(options, "dry_run", "dry-run"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "repo:     %s/%s", output.Org, output.Repo)
	core.Print(nil, "audits:   %d", len(output.Audits))
	core.Print(nil, "created:  %d", len(output.Created))
	core.Print(nil, "existing: %d", len(output.Existing))
	if len(output.Closed) > 0 {
		core.Print(nil, "closed:   %d", len(output.Closed))
	}
	for _, issue := range output.Created {
		core.Print(nil, "  created:  #%d %s", issue.Number, issue.Title)
	}
	for _, issue := range output.Existing {
		core.Print(nil, "  existing: #%d %s", issue.Number, issue.Title)
	}
	if len(output.Audits) == 0 {
		core.Print(nil, "no audit issues")
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) pipelineAudit(ctx context.Context, input PipelineAuditInput) (PipelineAuditOutput, error) {
	if input.Repo == "" {
		return PipelineAuditOutput{}, core.E("pipelineAudit", "repo is required", nil)
	}
	if s.forgeToken == "" {
		return PipelineAuditOutput{}, core.E("pipelineAudit", "no Forge token configured", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}

	issues, err := s.pipelineListIssues(ctx, input.Org, input.Repo, "open")
	if err != nil {
		return PipelineAuditOutput{}, err
	}

	output := PipelineAuditOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
	}

	existingByTitle := make(map[string]PipelineIssueRef)
	for _, issue := range issues {
		if pipelineIssueState(issue) != "open" || pipelineIssueIsAudit(issue) || pipelineIssueIsEpic(issue) {
			continue
		}
		key := pipelineAuditExistingKey(issue)
		if key == "" {
			continue
		}
		existingByTitle[key] = pipelineIssueRefFromRecord(issue)
	}

	for _, issue := range issues {
		if !pipelineIssueIsAudit(issue) {
			continue
		}
		output.Audits = append(output.Audits, pipelineIssueRefFromRecord(issue))

		findings := pipelineAuditFindings(issue)
		if len(findings) == 0 {
			findings = []string{issue.Title}
		}

		linked := make([]PipelineIssueRef, 0, len(findings))
		for _, finding := range findings {
			title := pipelineAuditImplementationTitle(input.Repo, issue, finding)
			key := core.Concat(core.Sprint(issue.Number), ":", title)
			if existing, ok := existingByTitle[key]; ok {
				output.Existing = append(output.Existing, existing)
				linked = append(linked, existing)
				continue
			}

			labels := pipelineAuditLabels(issue)
			body := pipelineAuditImplementationBody(issue, finding)
			if input.DryRun {
				ref := PipelineIssueRef{Title: title, State: "planned", Labels: labels}
				output.Created = append(output.Created, ref)
				linked = append(linked, ref)
				continue
			}

			labelIDs := s.resolveLabelIDs(ctx, input.Org, input.Repo, labels)
			created, createErr := s.createIssue(ctx, input.Org, input.Repo, title, body, labelIDs)
			if createErr != nil {
				return PipelineAuditOutput{}, core.E("pipelineAudit", core.Concat("failed to create implementation issue for audit #", core.Sprint(issue.Number)), createErr)
			}

			ref := PipelineIssueRef{
				Number: created.Number,
				Title:  created.Title,
				URL:    created.URL,
				State:  "open",
				Labels: labels,
			}
			existingByTitle[key] = ref
			output.Created = append(output.Created, ref)
			linked = append(linked, ref)
		}

		if input.DryRun || len(linked) == 0 {
			continue
		}

		s.commentOnIssue(ctx, input.Org, input.Repo, issue.Number, pipelineAuditLinkedComment(linked))
		if closeErr := s.pipelinePatchIssue(ctx, input.Org, input.Repo, issue.Number, map[string]any{"state": "closed"}); closeErr != nil {
			return PipelineAuditOutput{}, closeErr
		}
		output.Closed = append(output.Closed, issue.Number)
	}

	return output, nil
}

func (s *PrepSubsystem) pipelineListOrgRepos(ctx context.Context, org string) ([]pipelineRepoRecord, error) {
	url := core.Sprintf("%s/api/v1/orgs/%s/repos?limit=100&page=1", s.forgeURL, org)
	result := HTTPGet(ctx, url, s.forgeToken, "token")
	if !result.OK {
		return nil, core.E("pipelineListOrgRepos", core.Concat("failed to list repos for ", org), nil)
	}

	var repos []pipelineRepoRecord
	if err := pipelineDecodeJSON(result.Value.(string), &repos, "pipelineListOrgRepos", "failed to decode repo list"); err != nil {
		return nil, err
	}
	return repos, nil
}

func (s *PrepSubsystem) pipelineGetRepo(ctx context.Context, org, repo string) (pipelineRepoRecord, error) {
	url := core.Sprintf("%s/api/v1/repos/%s/%s", s.forgeURL, org, repo)
	result := HTTPGet(ctx, url, s.forgeToken, "token")
	if !result.OK {
		return pipelineRepoRecord{}, core.E("pipelineGetRepo", core.Concat("failed to read repo ", repo), nil)
	}

	var record pipelineRepoRecord
	if err := pipelineDecodeJSON(result.Value.(string), &record, "pipelineGetRepo", "failed to decode repo"); err != nil {
		return pipelineRepoRecord{}, err
	}
	return record, nil
}

func (s *PrepSubsystem) pipelineListIssues(ctx context.Context, org, repo, state string) ([]pipelineIssueRecord, error) {
	if state == "" {
		state = "open"
	}

	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues?state=%s&limit=100&type=issues", s.forgeURL, org, repo, state)
	result := HTTPGet(ctx, url, s.forgeToken, "token")
	if !result.OK {
		return nil, core.E("pipelineListIssues", core.Concat("failed to list issues for ", repo), nil)
	}

	var issues []pipelineIssueRecord
	if err := pipelineDecodeJSON(result.Value.(string), &issues, "pipelineListIssues", "failed to decode issue list"); err != nil {
		return nil, err
	}
	return issues, nil
}

func (s *PrepSubsystem) pipelineGetIssue(ctx context.Context, org, repo string, number int) (pipelineIssueRecord, error) {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, org, repo, number)
	result := HTTPGet(ctx, url, s.forgeToken, "token")
	if !result.OK {
		return pipelineIssueRecord{}, core.E("pipelineGetIssue", core.Concat("failed to read issue #", core.Sprint(number)), nil)
	}

	var issue pipelineIssueRecord
	if err := pipelineDecodeJSON(result.Value.(string), &issue, "pipelineGetIssue", "failed to decode issue"); err != nil {
		return pipelineIssueRecord{}, err
	}
	return issue, nil
}

func (s *PrepSubsystem) pipelinePatchIssue(ctx context.Context, org, repo string, number int, payload map[string]any) error {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, org, repo, number)
	result := HTTPPatch(ctx, url, core.JSONMarshalString(payload), s.forgeToken, "token")
	if !result.OK {
		return core.E("pipelinePatchIssue", core.Concat("failed to update issue #", core.Sprint(number)), nil)
	}
	return nil
}

func pipelineDecodeJSON(data string, target any, errorName, message string) error {
	parseResult := core.JSONUnmarshalString(data, target)
	if !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return core.E(errorName, message, err)
	}
	return nil
}

func pipelineIssueRefFromRecord(issue pipelineIssueRecord) PipelineIssueRef {
	return PipelineIssueRef{
		Number: issue.Number,
		Title:  issue.Title,
		State:  pipelineIssueState(issue),
		URL:    issue.HTMLURL,
		Labels: pipelineIssueLabelNames(issue),
	}
}

func pipelineIssueState(issue pipelineIssueRecord) string {
	state := core.Lower(core.Trim(issue.State))
	if state == "" {
		return "open"
	}
	return state
}

func pipelineIssueLabelNames(issue pipelineIssueRecord) []string {
	names := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		if label.Name != "" {
			names = append(names, label.Name)
		}
	}
	return names
}

func pipelineIssueHasLabel(issue pipelineIssueRecord, want string) bool {
	for _, name := range pipelineIssueLabelNames(issue) {
		if core.Lower(name) == core.Lower(want) {
			return true
		}
	}
	return false
}

func pipelineIssueIsAudit(issue pipelineIssueRecord) bool {
	title := core.Lower(issue.Title)
	return pipelineIssueHasLabel(issue, "audit") || core.Contains(title, "[audit]") || core.HasPrefix(title, "audit:")
}

func pipelineIssueIsEpic(issue pipelineIssueRecord) bool {
	return pipelineIssueHasLabel(issue, "epic") || regexp.MustCompile(`(?m)^\s*-\s*\[[ xX]\]\s*#\d+`).MatchString(issue.Body)
}

func pipelineIssueIsImplementationCandidate(issue pipelineIssueRecord) bool {
	if pipelineIssueState(issue) != "open" || pipelineIssueIsAudit(issue) || pipelineIssueIsEpic(issue) {
		return false
	}
	if len(issue.PullRequest) > 0 {
		return false
	}
	return !core.Contains(issue.Body, "Parent: #")
}

func pipelineAuditFindings(issue pipelineIssueRecord) []string {
	lines := core.Split(issue.Body, "\n")
	findings := []string{}
	for _, line := range lines {
		trimmed := core.Trim(line)
		if trimmed == "" || core.HasPrefix(trimmed, "#") {
			continue
		}

		match := pipelineBulletPattern.FindStringSubmatch(trimmed)
		if len(match) == 2 {
			findings = append(findings, pipelineFindingSummary(match[1]))
		}
	}

	if len(findings) > 0 {
		return findings
	}

	paragraphs := core.Split(issue.Body, "\n\n")
	for _, paragraph := range paragraphs {
		summary := pipelineFindingSummary(paragraph)
		if summary == "" || summary == issue.Title {
			continue
		}
		findings = append(findings, summary)
		break
	}
	return findings
}

func pipelineFindingSummary(value string) string {
	summary := pipelineWhitespacePattern.ReplaceAllString(core.Trim(value), " ")
	summary = regexp.MustCompile("`").ReplaceAllString(summary, "")
	if summary == "" {
		return ""
	}
	if len(summary) > 96 {
		summary = core.Concat(summary[:93], "...")
	}
	return summary
}

func pipelineAuditIssueType(issue pipelineIssueRecord) string {
	haystack := core.Lower(core.Concat(issue.Title, " ", issue.Body, " ", core.JSONMarshalString(pipelineIssueLabelNames(issue))))
	switch {
	case core.Contains(haystack, "security"), core.Contains(haystack, "owasp"), core.Contains(haystack, "auth"), core.Contains(haystack, "sanitize"), core.Contains(haystack, "validation"):
		return "security"
	case core.Contains(haystack, "test"), core.Contains(haystack, "coverage"):
		return "testing"
	case core.Contains(haystack, "perf"), core.Contains(haystack, "performance"):
		return "performance"
	case core.Contains(haystack, "doc"):
		return "docs"
	default:
		return "quality"
	}
}

func pipelineAuditSeverity(issue pipelineIssueRecord) string {
	haystack := core.Lower(core.Concat(issue.Title, " ", issue.Body))
	switch {
	case core.Contains(haystack, "critical"):
		return "critical"
	case core.Contains(haystack, "high"):
		return "high"
	case core.Contains(haystack, "medium"):
		return "medium"
	case core.Contains(haystack, "low"):
		return "low"
	default:
		return ""
	}
}

func pipelineAuditLabels(issue pipelineIssueRecord) []string {
	labels := []string{"agentic", pipelineAuditIssueType(issue)}
	if severity := pipelineAuditSeverity(issue); severity != "" {
		labels = append(labels, severity)
	}
	return labels
}

func pipelineAuditImplementationTitle(repo string, issue pipelineIssueRecord, finding string) string {
	issueType := pipelineAuditIssueType(issue)
	summary := pipelineFindingSummary(finding)
	if summary == "" {
		summary = issue.Title
	}
	return core.Sprintf("%s(%s): %s", issueType, repo, summary)
}

func pipelineAuditImplementationBody(issue pipelineIssueRecord, finding string) string {
	builder := core.NewBuilder()
	builder.WriteString(core.Sprintf("Parent audit: #%d\n\n", issue.Number))
	builder.WriteString("## Finding\n\n")
	builder.WriteString(pipelineFindingSummary(finding))
	builder.WriteString("\n\n## Acceptance Criteria\n\n")
	builder.WriteString("- [ ] Implement the audit finding\n")
	builder.WriteString("- [ ] Update tests or validation where needed\n")
	builder.WriteString("- [ ] Link the resulting PR back to this issue\n")
	return builder.String()
}

func pipelineAuditExistingKey(issue pipelineIssueRecord) string {
	match := regexp.MustCompile(`Parent audit:\s*#(\d+)`).FindStringSubmatch(issue.Body)
	if len(match) != 2 {
		return ""
	}
	return core.Concat(match[1], ":", issue.Title)
}

func pipelineAuditLinkedComment(linked []PipelineIssueRef) string {
	builder := core.NewBuilder()
	builder.WriteString("Implementation issues created:\n")
	for _, issue := range linked {
		if issue.Number > 0 {
			builder.WriteString(core.Sprintf("- #%d %s\n", issue.Number, issue.Title))
			continue
		}
		builder.WriteString(core.Sprintf("- %s\n", issue.Title))
	}
	return builder.String()
}
