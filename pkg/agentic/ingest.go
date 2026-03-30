// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

func agentHomeDir() string {
	return HomeDir()
}

// ingestFindings reads the agent output log and creates issues via the API
// for scan/audit results. Only runs for conventions and security templates.
func (s *PrepSubsystem) ingestFindings(wsDir string) {
	result := ReadStatusResult(wsDir)
	st, ok := workspaceStatusValue(result)
	if !ok || st.Status != "completed" {
		return
	}

	// Read the log file
	logFiles := workspaceLogFiles(wsDir)
	if len(logFiles) == 0 {
		return
	}

	r := fs.Read(logFiles[0])
	if !r.OK || len(r.Value.(string)) < 100 {
		return
	}

	body := r.Value.(string)

	// Skip quota errors
	if core.Contains(body, "QUOTA_EXHAUSTED") || core.Contains(body, "QuotaError") {
		return
	}

	// Only ingest if there are actual findings (file:line references)
	findings := countFileRefs(body)
	if findings < 2 {
		return // No meaningful findings
	}

	// Determine issue type from the template used
	issueType := "task"
	priority := "normal"
	if core.Contains(body, "security") || core.Contains(body, "Security") {
		issueType = "bug"
		priority = "high"
	}

	// Create a single issue per repo with all findings in the body
	title := core.Sprintf("Scan findings for %s (%d items)", st.Repo, findings)

	// Truncate body to reasonable size for issue description
	description := body
	if len(description) > 10000 {
		description = core.Concat(description[:10000], "\n\n... (truncated, see full log in workspace)")
	}

	s.createIssueViaAPI(st.Repo, title, description, issueType, priority, "scan")
}

// countFileRefs counts file:line references in the output (indicates real findings)
func countFileRefs(body string) int {
	count := 0
	for i := 0; i < len(body)-5; i++ {
		if body[i] == '`' {
			// Look for pattern: `file.go:123`
			j := i + 1
			for j < len(body) && body[j] != '`' && j-i < 100 {
				j++
			}
			if j < len(body) && body[j] == '`' {
				ref := body[i+1 : j]
				if core.Contains(ref, ".go:") || core.Contains(ref, ".php:") {
					count++
				}
			}
		}
	}
	return count
}

// createIssueViaAPI posts an issue to the lthn.sh API
func (s *PrepSubsystem) createIssueViaAPI(repo, title, description, issueType, priority, source string) {
	if s.brainKey == "" {
		return
	}

	// Read the agent API key from file
	r := fs.Read(core.JoinPath(agentHomeDir(), ".claude", "agent-api.key"))
	if !r.OK {
		return
	}
	apiKey := core.Trim(r.Value.(string))

	payload := core.JSONMarshalString(map[string]string{
		"title":       title,
		"description": description,
		"type":        issueType,
		"priority":    priority,
		"reporter":    "cladius",
	})

	HTTPPost(context.Background(), core.Concat(s.brainURL, "/v1/issues"), payload, apiKey, "Bearer")
}
