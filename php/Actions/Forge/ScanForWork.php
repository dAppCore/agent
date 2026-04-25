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
use Core\Mod\Agentic\Pipeline\ForgejoMetaReader;
use Core\Mod\Agentic\Pipeline\MetaReader;
use Core\Mod\Agentic\Services\ForgejoService;

/**
 * Scan Forgejo for epic issues and identify unchecked children that need coding.
 *
 * Reads structural epic metadata and issue state through MetaReader and
 * returns work items for any unchecked child issue that has no linked PR.
 *
 * Usage:
 *   $workItems = ScanForWork::run('core', 'app');
 */
class ScanForWork
{
    use Action;

    public function __construct(
        private ?MetaReader $metaReader = null,
    ) {}

    /**
     * Scan a repository for actionable work from epic issues.
     *
     * @return array<int, array{
     *     epic_number: int,
     *     issue_number: int,
     *     issue_title: string,
     *     issue_state: string,
     *     issue_labels: array<int, string>,
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
        $metaReader = $this->resolveMetaReader($owner, $repo);

        $epics = $forge->listIssues($owner, $repo, 'open', 'epic');

        if ($epics === []) {
            return [];
        }

        $workItems = [];

        foreach ($epics as $epic) {
            $epicNumber = (int) ($epic['number'] ?? 0);

            if ($epicNumber === 0) {
                continue;
            }

            $epicMeta = $metaReader->getEpicMeta($epicNumber);

            foreach ($epicMeta->children as $childMeta) {
                if ($childMeta->checkedBool) {
                    continue;
                }

                if ($childMeta->linkedPrNumberOrNull !== null) {
                    continue;
                }

                if ($childMeta->state !== 'open') {
                    continue;
                }

                $issueState = $metaReader->getIssueState($childMeta->issueId);

                if ($issueState->state !== 'open') {
                    continue;
                }

                $workItems[] = [
                    'epic_number' => $epicNumber,
                    'issue_number' => $childMeta->issueId,
                    'issue_title' => $issueState->title,
                    'issue_state' => $issueState->state,
                    'issue_labels' => $issueState->labels,
                    'assignee' => $issueState->assignee,
                    'repo_owner' => $owner,
                    'repo_name' => $repo,
                    'needs_coding' => true,
                    'has_pr' => false,
                ];
            }
        }

        return $workItems;
    }

    private function resolveMetaReader(string $owner, string $repo): MetaReader
    {
        if ($this->metaReader instanceof MetaReader) {
            return $this->metaReader;
        }

        /** @var MetaReader $metaReader */
        $metaReader = app()->makeWith(ForgejoMetaReader::class, [
            'owner' => $owner,
            'repo' => $repo,
        ]);

        return $metaReader;
    }
}
