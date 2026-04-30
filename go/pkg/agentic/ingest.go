// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

func (s *PrepSubsystem) ingestFindings(workspaceDir string) {
	statusResult := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(statusResult)
	if !ok || workspaceStatus.Status != "completed" {
		return
	}

	logFiles := workspaceLogFiles(workspaceDir)
	if len(logFiles) == 0 {
		return
	}

	logResult := fs.Read(logFiles[0])
	if !logResult.OK || len(logResult.Value.(string)) < 100 {
		return
	}

	logBody := logResult.Value.(string)

	if core.Contains(logBody, "QUOTA_EXHAUSTED") || core.Contains(logBody, "QuotaError") {
		return
	}

	findings := countFileRefs(logBody)
	if findings < 2 {
		return
	}

	issueType := "task"
	priority := "normal"
	if core.Contains(logBody, "security") || core.Contains(logBody, "Security") {
		issueType = "bug"
		priority = "high"
	}

	title := core.Sprintf("Scan findings for %s (%d items)", workspaceStatus.Repo, findings)

	issueDescription := logBody
	if len(issueDescription) > 10000 {
		issueDescription = core.Concat(issueDescription[:10000], "\n\n... (truncated, see full log in workspace)")
	}

	s.createIssueViaAPI(title, issueDescription, issueType, priority)
}

func countFileRefs(body string) int {
	count := 0
	for i := 0; i < len(body)-5; i++ {
		if body[i] == '`' {
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

func (s *PrepSubsystem) createIssueViaAPI(title, description, issueType, priority string) {
	if s.brainKey == "" {
		return
	}

	apiKeyResult := fs.Read(core.JoinPath(HomeDir(), ".claude", "agent-api.key"))
	if !apiKeyResult.OK {
		return
	}
	apiKey := core.Trim(apiKeyResult.Value.(string))

	issuePayload := core.JSONMarshalString(map[string]string{
		"title":       title,
		"description": description,
		"type":        issueType,
		"priority":    priority,
		"reporter":    "cladius",
	})

	HTTPPost(context.Background(), core.Concat(s.brainURL, "/v1/issues"), issuePayload, apiKey, "Bearer")
}
