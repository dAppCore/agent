// cmd_prep.go implements the `workspace prep` command.
//
// Prepares an agent workspace with wiki KB, protocol specs, a TODO from a
// Forge issue, and vector-recalled context from OpenBrain. All output goes
// to .core/ in the current directory, matching the convention used by
// KBConfig (go-scm) and build/release config.

package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"forge.lthn.ai/core/agent/pkg/lifecycle"
	"forge.lthn.ai/core/cli/pkg/cli"
	coreio "forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-scm/forge"
)

var (
	prepRepo      string
	prepIssue     int
	prepOrg       string
	prepOutput    string
	prepSpecsPath string
	prepDryRun    bool
)

func addPrepCommands(parent *cli.Command) {
	prepCmd := &cli.Command{
		Use:   "prep",
		Short: "Prepare agent workspace with wiki KB, specs, TODO, and vector context",
		Long: `Fetches wiki pages from Forge, copies protocol specs, generates a task
file from a Forge issue, and queries OpenBrain for relevant context.
All output is written to .core/ in the current directory.`,
		RunE: runPrep,
	}

	prepCmd.Flags().StringVar(&prepRepo, "repo", "", "Forge repo name (e.g. go-ai)")
	prepCmd.Flags().IntVar(&prepIssue, "issue", 0, "Issue number to build TODO from")
	prepCmd.Flags().StringVar(&prepOrg, "org", "core", "Forge organisation")
	prepCmd.Flags().StringVar(&prepOutput, "output", "", "Output directory (default: ./.core)")
	prepCmd.Flags().StringVar(&prepSpecsPath, "specs-path", "", "Path to specs dir")
	prepCmd.Flags().BoolVar(&prepDryRun, "dry-run", false, "Preview without writing files")
	_ = prepCmd.MarkFlagRequired("repo")

	parent.AddCommand(prepCmd)
}

func runPrep(cmd *cli.Command, args []string) error {
	ctx := context.Background()

	// Resolve output directory
	outputDir := prepOutput
	if outputDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return cli.Err("failed to get working directory")
		}
		outputDir = filepath.Join(cwd, ".core")
	}

	// Resolve specs path
	specsPath := prepSpecsPath
	if specsPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			specsPath = filepath.Join(home, "Code", "specs")
		}
	}

	// Resolve Forge connection
	forgeURL, forgeToken, err := forge.ResolveConfig("", "")
	if err != nil {
		return log.E("workspace.prep", "failed to resolve Forge config", err)
	}
	if forgeToken == "" {
		return log.E("workspace.prep", "no Forge token configured — set FORGE_TOKEN or run: core forge login", nil)
	}

	cli.Print("Preparing workspace for %s/%s\n", cli.ValueStyle.Render(prepOrg), cli.ValueStyle.Render(prepRepo))
	cli.Print("Output: %s\n", cli.DimStyle.Render(outputDir))

	if prepDryRun {
		cli.Print("%s No files will be written.\n", cli.WarningStyle.Render("[DRY RUN]"))
	}
	fmt.Println()

	// Create output directory structure
	if !prepDryRun {
		if err := coreio.Local.EnsureDir(filepath.Join(outputDir, "kb")); err != nil {
			return log.E("workspace.prep", "failed to create kb directory", err)
		}
		if err := coreio.Local.EnsureDir(filepath.Join(outputDir, "specs")); err != nil {
			return log.E("workspace.prep", "failed to create specs directory", err)
		}
	}

	// Step 1: Pull wiki pages
	wikiCount, err := prepPullWiki(ctx, forgeURL, forgeToken, prepOrg, prepRepo, outputDir, prepDryRun)
	if err != nil {
		cli.Print("%s wiki: %v\n", cli.WarningStyle.Render("warn"), err)
	}

	// Step 2: Copy spec files
	specsCount := prepCopySpecs(specsPath, outputDir, prepDryRun)

	// Step 3: Generate TODO from issue
	var issueTitle, issueBody string
	if prepIssue > 0 {
		issueTitle, issueBody, err = prepGenerateTodo(ctx, forgeURL, forgeToken, prepOrg, prepRepo, prepIssue, outputDir, prepDryRun)
		if err != nil {
			cli.Print("%s todo: %v\n", cli.WarningStyle.Render("warn"), err)
			prepGenerateTodoSkeleton(prepOrg, prepRepo, outputDir, prepDryRun)
		}
	} else {
		prepGenerateTodoSkeleton(prepOrg, prepRepo, outputDir, prepDryRun)
	}

	// Step 4: Generate context from OpenBrain
	contextCount := prepGenerateContext(ctx, prepRepo, issueTitle, issueBody, outputDir, prepDryRun)

	// Summary
	fmt.Println()
	prefix := ""
	if prepDryRun {
		prefix = "[DRY RUN] "
	}
	cli.Print("%s%s\n", prefix, cli.SuccessStyle.Render("Workspace prep complete:"))
	cli.Print("  Wiki pages:  %s\n", cli.ValueStyle.Render(fmt.Sprintf("%d", wikiCount)))
	cli.Print("  Spec files:  %s\n", cli.ValueStyle.Render(fmt.Sprintf("%d", specsCount)))
	if issueTitle != "" {
		cli.Print("  TODO:        %s\n", cli.ValueStyle.Render(fmt.Sprintf("from issue #%d", prepIssue)))
	} else {
		cli.Print("  TODO:        %s\n", cli.DimStyle.Render("skeleton"))
	}
	cli.Print("  Context:     %s\n", cli.ValueStyle.Render(fmt.Sprintf("%d memories", contextCount)))

	return nil
}

// --- Step 1: Pull wiki pages from Forge API ---

type wikiPageRef struct {
	Title  string `json:"title"`
	SubURL string `json:"sub_url"`
}

type wikiPageContent struct {
	ContentBase64 string `json:"content_base64"`
}

func prepPullWiki(ctx context.Context, forgeURL, token, org, repo, outputDir string, dryRun bool) (int, error) {
	cli.Print("Fetching wiki pages for %s/%s...\n", org, repo)

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/pages", forgeURL, org, repo)
	resp, err := forgeGet(ctx, endpoint, token)
	if err != nil {
		return 0, log.E("workspace.prep.wiki", "API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		cli.Print("  %s No wiki found for %s\n", cli.WarningStyle.Render("warn"), repo)
		if !dryRun {
			content := fmt.Sprintf("# No wiki found for %s\n\nThis repo has no wiki pages on Forge.\n", repo)
			_ = coreio.Local.Write(filepath.Join(outputDir, "kb", "README.md"), content)
		}
		return 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		return 0, log.E("workspace.prep.wiki", fmt.Sprintf("API error: %d", resp.StatusCode), nil)
	}

	var pages []wikiPageRef
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return 0, log.E("workspace.prep.wiki", "failed to decode pages", err)
	}

	if len(pages) == 0 {
		cli.Print("  %s Wiki exists but has no pages.\n", cli.WarningStyle.Render("warn"))
		return 0, nil
	}

	count := 0
	for _, page := range pages {
		title := page.Title
		if title == "" {
			title = "Untitled"
		}
		subURL := page.SubURL
		if subURL == "" {
			subURL = title
		}

		if dryRun {
			cli.Print("  [would fetch] %s\n", title)
			count++
			continue
		}

		pageEndpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/page/%s",
			forgeURL, org, repo, url.PathEscape(subURL))
		pageResp, err := forgeGet(ctx, pageEndpoint, token)
		if err != nil || pageResp.StatusCode != http.StatusOK {
			cli.Print("  %s Failed to fetch: %s\n", cli.WarningStyle.Render("warn"), title)
			if pageResp != nil {
				pageResp.Body.Close()
			}
			continue
		}

		var pageData wikiPageContent
		if err := json.NewDecoder(pageResp.Body).Decode(&pageData); err != nil {
			pageResp.Body.Close()
			continue
		}
		pageResp.Body.Close()

		if pageData.ContentBase64 == "" {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(pageData.ContentBase64)
		if err != nil {
			continue
		}

		filename := sanitiseFilename(title) + ".md"
		_ = coreio.Local.Write(filepath.Join(outputDir, "kb", filename), string(decoded))
		cli.Print("  %s\n", title)
		count++
	}

	cli.Print("  %d wiki page(s) saved to kb/\n", count)
	return count, nil
}

// --- Step 2: Copy protocol spec files ---

func prepCopySpecs(specsPath, outputDir string, dryRun bool) int {
	cli.Print("Copying spec files...\n")

	specFiles := []string{"AGENT_CONTEXT.md", "TASK_PROTOCOL.md"}
	count := 0

	for _, file := range specFiles {
		source := filepath.Join(specsPath, file)
		if !coreio.Local.IsFile(source) {
			cli.Print("  %s Not found: %s\n", cli.WarningStyle.Render("warn"), source)
			continue
		}

		if dryRun {
			cli.Print("  [would copy] %s\n", file)
			count++
			continue
		}

		content, err := coreio.Local.Read(source)
		if err != nil {
			cli.Print("  %s Failed to read: %s\n", cli.WarningStyle.Render("warn"), file)
			continue
		}

		dest := filepath.Join(outputDir, "specs", file)
		if err := coreio.Local.Write(dest, content); err != nil {
			cli.Print("  %s Failed to write: %s\n", cli.WarningStyle.Render("warn"), file)
			continue
		}

		cli.Print("  %s\n", file)
		count++
	}

	cli.Print("  %d spec file(s) copied.\n", count)
	return count
}

// --- Step 3: Generate TODO from Forge issue ---

type forgeIssue struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func prepGenerateTodo(ctx context.Context, forgeURL, token, org, repo string, issueNum int, outputDir string, dryRun bool) (string, string, error) {
	cli.Print("Generating TODO from issue #%d...\n", issueNum)

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", forgeURL, org, repo, issueNum)
	resp, err := forgeGet(ctx, endpoint, token)
	if err != nil {
		return "", "", log.E("workspace.prep.todo", "issue API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", log.E("workspace.prep.todo", fmt.Sprintf("failed to fetch issue #%d: %d", issueNum, resp.StatusCode), nil)
	}

	var issue forgeIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return "", "", log.E("workspace.prep.todo", "failed to decode issue", err)
	}

	title := issue.Title
	if title == "" {
		title = "Untitled"
	}

	objective := extractObjective(issue.Body)
	checklist := extractChecklist(issue.Body)

	var b strings.Builder
	fmt.Fprintf(&b, "# TASK: %s\n\n", title)
	fmt.Fprintf(&b, "**Status:** ready\n")
	fmt.Fprintf(&b, "**Source:** %s/%s/%s/issues/%d\n", forgeURL, org, repo, issueNum)
	fmt.Fprintf(&b, "**Created:** %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**Repo:** %s/%s\n", org, repo)
	b.WriteString("\n---\n\n")

	fmt.Fprintf(&b, "## Objective\n\n%s\n", objective)
	b.WriteString("\n---\n\n")

	b.WriteString("## Acceptance Criteria\n\n")
	if len(checklist) > 0 {
		for _, item := range checklist {
			fmt.Fprintf(&b, "- [ ] %s\n", item)
		}
	} else {
		b.WriteString("_No checklist items found in issue. Agent should define acceptance criteria._\n")
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Implementation Checklist\n\n")
	b.WriteString("_To be filled by the agent during planning._\n")
	b.WriteString("\n---\n\n")

	b.WriteString("## Notes\n\n")
	b.WriteString("Full issue body preserved below for reference.\n\n")
	b.WriteString("<details>\n<summary>Original Issue</summary>\n\n")
	b.WriteString(issue.Body)
	b.WriteString("\n\n</details>\n")

	if dryRun {
		cli.Print("  [would write] todo.md from: %s\n", title)
	} else {
		if err := coreio.Local.Write(filepath.Join(outputDir, "todo.md"), b.String()); err != nil {
			return title, issue.Body, log.E("workspace.prep.todo", "failed to write todo.md", err)
		}
		cli.Print("  todo.md generated from: %s\n", title)
	}

	return title, issue.Body, nil
}

func prepGenerateTodoSkeleton(org, repo, outputDir string, dryRun bool) {
	var b strings.Builder
	b.WriteString("# TASK: [Define task]\n\n")
	fmt.Fprintf(&b, "**Status:** ready\n")
	fmt.Fprintf(&b, "**Created:** %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**Repo:** %s/%s\n", org, repo)
	b.WriteString("\n---\n\n")
	b.WriteString("## Objective\n\n_Define the objective._\n")
	b.WriteString("\n---\n\n")
	b.WriteString("## Acceptance Criteria\n\n- [ ] _Define criteria_\n")
	b.WriteString("\n---\n\n")
	b.WriteString("## Implementation Checklist\n\n_To be filled by the agent._\n")

	if dryRun {
		cli.Print("  [would write] todo.md skeleton\n")
	} else {
		_ = coreio.Local.Write(filepath.Join(outputDir, "todo.md"), b.String())
		cli.Print("  todo.md skeleton generated (no --issue provided)\n")
	}
}

// --- Step 4: Generate context from OpenBrain ---

func prepGenerateContext(ctx context.Context, repo, issueTitle, issueBody, outputDir string, dryRun bool) int {
	cli.Print("Querying vector DB for context...\n")

	apiURL := os.Getenv("CORE_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8000"
	}
	apiToken := os.Getenv("CORE_API_TOKEN")

	client := lifecycle.NewClient(apiURL, apiToken)

	// Query 1: Repo-specific knowledge
	repoResult, err := client.Recall(ctx, lifecycle.RecallRequest{
		Query:   "How does " + repo + " work? Architecture and key interfaces.",
		TopK:    10,
		Project: repo,
	})
	if err != nil {
		cli.Print("  %s BrainService unavailable: %v\n", cli.WarningStyle.Render("warn"), err)
		writeBrainUnavailable(repo, outputDir, dryRun)
		return 0
	}

	repoMemories := repoResult.Memories
	repoScores := repoResult.Scores

	// Query 2: Issue-specific context
	var issueMemories []lifecycle.Memory
	var issueScores map[string]float64
	if issueTitle != "" {
		query := issueTitle
		if len(issueBody) > 500 {
			query += " " + issueBody[:500]
		} else if issueBody != "" {
			query += " " + issueBody
		}

		issueResult, err := client.Recall(ctx, lifecycle.RecallRequest{
			Query: query,
			TopK:  5,
		})
		if err == nil {
			issueMemories = issueResult.Memories
			issueScores = issueResult.Scores
		}
	}

	totalMemories := len(repoMemories) + len(issueMemories)

	var b strings.Builder
	fmt.Fprintf(&b, "# Agent Context — %s\n\n", repo)
	b.WriteString("> Auto-generated by `core workspace prep`. Query the vector DB for more.\n\n")

	b.WriteString("## Repo Knowledge\n\n")
	if len(repoMemories) > 0 {
		for i, mem := range repoMemories {
			score := repoScores[mem.ID]
			project := mem.Project
			if project == "" {
				project = "unknown"
			}
			memType := mem.Type
			if memType == "" {
				memType = "memory"
			}
			fmt.Fprintf(&b, "### %d. %s [%s] (score: %.3f)\n\n", i+1, project, memType, score)
			fmt.Fprintf(&b, "%s\n\n", mem.Content)
		}
	} else {
		b.WriteString("_No repo-specific memories found. The vector DB may not have been seeded for this repo._\n\n")
	}

	b.WriteString("## Task-Relevant Context\n\n")
	if len(issueMemories) > 0 {
		for i, mem := range issueMemories {
			score := issueScores[mem.ID]
			project := mem.Project
			if project == "" {
				project = "unknown"
			}
			memType := mem.Type
			if memType == "" {
				memType = "memory"
			}
			fmt.Fprintf(&b, "### %d. %s [%s] (score: %.3f)\n\n", i+1, project, memType, score)
			fmt.Fprintf(&b, "%s\n\n", mem.Content)
		}
	} else if issueTitle != "" {
		b.WriteString("_No task-relevant memories found._\n\n")
	} else {
		b.WriteString("_No issue provided — skipped task-specific recall._\n\n")
	}

	if dryRun {
		cli.Print("  [would write] context.md with %d memories\n", totalMemories)
	} else {
		_ = coreio.Local.Write(filepath.Join(outputDir, "context.md"), b.String())
		cli.Print("  context.md generated with %d memories\n", totalMemories)
	}

	return totalMemories
}

func writeBrainUnavailable(repo, outputDir string, dryRun bool) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Agent Context — %s\n\n", repo)
	b.WriteString("> Vector DB was unavailable when this workspace was prepared.\n")
	b.WriteString("> Run `core workspace prep` again once Ollama/Qdrant are reachable.\n")

	if !dryRun {
		_ = coreio.Local.Write(filepath.Join(outputDir, "context.md"), b.String())
	}
}

// --- Helpers ---

func forgeGet(ctx context.Context, endpoint, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)

func sanitiseFilename(title string) string {
	return nonAlphanumeric.ReplaceAllString(title, "-")
}

func extractObjective(body string) string {
	if body == "" {
		return "_No description provided._"
	}
	parts := strings.SplitN(body, "\n\n", 2)
	first := strings.TrimSpace(parts[0])
	if len(first) > 500 {
		return first[:497] + "..."
	}
	return first
}

func extractChecklist(body string) []string {
	re := regexp.MustCompile(`- \[[ xX]\] (.+)`)
	matches := re.FindAllStringSubmatch(body, -1)
	var items []string
	for _, m := range matches {
		items = append(items, strings.TrimSpace(m[1]))
	}
	return items
}
