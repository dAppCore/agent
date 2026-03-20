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
use Core\Mod\Agentic\Services\ForgejoService;

/**
 * Scan Forgejo for epic issues and identify unchecked children that need coding.
 *
 * Parses epic issue bodies for checklist syntax (`- [ ] #N` / `- [x] #N`),
 * cross-references with open pull requests, and returns structured work items
 * for any unchecked child issue that has no linked PR.
 *
 * Usage:
 *   $workItems = ScanForWork::run('core', 'app');
 */
class ScanForWork
{
    use Action;

    /**
     * Scan a repository for actionable work from epic issues.
     *
     * @return array<int, array{
     *     epic_number: int,
     *     issue_number: int,
     *     issue_title: string,
     *     issue_body: string,
     *     assignee: string|null,
     *     repo_owner: string,
     *     repo_name: string,
     *     needs_coding: bool,
     *     has_pr: bool,
     * }>
     */
    public function handle(string $owner, string $repo): array
    {
        $forge = app(ForgejoService::class);

        $epics = $forge->listIssues($owner, $repo, 'open', 'epic');

        if ($epics === []) {
            return [];
        }

        $pullRequests = $forge->listPullRequests($owner, $repo, 'all');
        $linkedIssues = $this->extractLinkedIssues($pullRequests);

        $workItems = [];

        foreach ($epics as $epic) {
            $checklist = $this->parseChecklist((string) ($epic['body'] ?? ''));

            foreach ($checklist as $item) {
                if ($item['checked']) {
                    continue;
                }

                if (in_array($item['number'], $linkedIssues, true)) {
                    continue;
                }

                $child = $forge->getIssue($owner, $repo, $item['number']);

                $assignee = null;
                if (! empty($child['assignees']) && is_array($child['assignees'])) {
                    $assignee = $child['assignees'][0]['login'] ?? null;
                }

                $workItems[] = [
                    'epic_number' => (int) $epic['number'],
                    'issue_number' => (int) $child['number'],
                    'issue_title' => (string) ($child['title'] ?? ''),
                    'issue_body' => (string) ($child['body'] ?? ''),
                    'assignee' => $assignee,
                    'repo_owner' => $owner,
                    'repo_name' => $repo,
                    'needs_coding' => true,
                    'has_pr' => false,
                ];
            }
        }

        return $workItems;
    }

    /**
     * Parse a checklist body into structured items.
     *
     * Matches lines like `- [ ] #2` (unchecked) and `- [x] #3` (checked).
     *
     * @return array<int, array{number: int, checked: bool}>
     */
    private function parseChecklist(string $body): array
    {
        $items = [];

        if (preg_match_all('/- \[([ xX])\] #(\d+)/', $body, $matches, PREG_SET_ORDER)) {
            foreach ($matches as $match) {
                $items[] = [
                    'number' => (int) $match[2],
                    'checked' => $match[1] !== ' ',
                ];
            }
        }

        return $items;
    }

    /**
     * Extract issue numbers referenced in PR bodies.
     *
     * Matches common linking patterns: "Closes #N", "Fixes #N", "Resolves #N",
     * and bare "#N" references.
     *
     * @param  array<int, array<string, mixed>>  $pullRequests
     * @return array<int, int>
     */
    private function extractLinkedIssues(array $pullRequests): array
    {
        $linked = [];

        foreach ($pullRequests as $pr) {
            $body = (string) ($pr['body'] ?? '');

            if (preg_match_all('/#(\d+)/', $body, $matches)) {
                foreach ($matches[1] as $number) {
                    $linked[] = (int) $number;
                }
            }
        }

        return array_unique($linked);
    }
}
