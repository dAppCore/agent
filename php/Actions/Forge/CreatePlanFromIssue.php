<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Forge;

use Core\Actions\Action;
use Core\Mod\Agentic\Actions\Plan\CreatePlan;
use Core\Mod\Agentic\Models\AgentPlan;

/**
 * Convert a Forgejo work item into an AgentPlan.
 *
 * Accepts the structured work item array produced by ScanForWork,
 * extracts checklist tasks from the issue body, and creates a plan
 * with a single phase. Returns an existing plan if one already
 * matches the same issue.
 *
 * Usage:
 *   $plan = CreatePlanFromIssue::run($workItem, $workspaceId);
 */
class CreatePlanFromIssue
{
    use Action;

    /**
     * @param  array{epic_number: int, issue_number: int, issue_title: string, issue_body: string, assignee: string|null, repo_owner: string, repo_name: string, needs_coding: bool, has_pr: bool}  $workItem
     */
    public function handle(array $workItem, int $workspaceId): AgentPlan
    {
        $issueNumber = (int) $workItem['issue_number'];
        $owner = (string) $workItem['repo_owner'];
        $repo = (string) $workItem['repo_name'];

        // Check for an existing plan for this issue (not archived)
        $existing = AgentPlan::where('status', '!=', AgentPlan::STATUS_ARCHIVED)
            ->whereJsonContains('metadata->issue_number', $issueNumber)
            ->whereJsonContains('metadata->repo_owner', $owner)
            ->whereJsonContains('metadata->repo_name', $repo)
            ->first();

        if ($existing !== null) {
            return $existing->load('agentPhases');
        }

        $tasks = $this->extractTasks((string) $workItem['issue_body']);

        $plan = CreatePlan::run([
            'title' => (string) $workItem['issue_title'],
            'slug' => "forge-{$owner}-{$repo}-{$issueNumber}",
            'description' => (string) $workItem['issue_body'],
            'phases' => [
                [
                    'name' => "Resolve issue #{$issueNumber}",
                    'description' => "Complete all tasks for issue #{$issueNumber}",
                    'tasks' => $tasks,
                ],
            ],
        ], $workspaceId);

        $plan->update([
            'metadata' => [
                'source' => 'forgejo',
                'epic_number' => (int) $workItem['epic_number'],
                'issue_number' => $issueNumber,
                'repo_owner' => $owner,
                'repo_name' => $repo,
                'assignee' => $workItem['assignee'] ?? null,
            ],
        ]);

        return $plan->load('agentPhases');
    }

    /**
     * Extract task names from markdown checklist items.
     *
     * Matches lines like `- [ ] Create picker UI` and returns
     * just the task name portion.
     *
     * @return array<int, string>
     */
    private function extractTasks(string $body): array
    {
        $tasks = [];

        if (preg_match_all('/- \[[ xX]\] (.+)/', $body, $matches)) {
            foreach ($matches[1] as $taskName) {
                $tasks[] = trim($taskName);
            }
        }

        return $tasks;
    }
}
