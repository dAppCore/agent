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
 * Evaluate and merge a Forgejo pull request when ready.
 *
 * Checks the PR state, mergeability, and CI status from MetaReader
 * before attempting the merge. Returns a result array describing the
 * outcome.
 *
 * Usage:
 *   $result = ManagePullRequest::run('core', 'app', 10);
 */
class ManagePullRequest
{
    use Action;

    public function __construct(
        private ?MetaReader $metaReader = null,
    ) {}

    /**
     * @return array{merged: bool, pr_number?: int, reason?: string}
     */
    public function handle(string $owner, string $repo, int $prNumber): array
    {
        $forge = app(ForgejoService::class);
        $metaReader = $this->resolveMetaReader($owner, $repo);
        $prMeta = $metaReader->getPRMeta($prNumber);

        if ($prMeta->state !== 'open') {
            return ['merged' => false, 'reason' => 'not_open'];
        }

        if ($prMeta->mergeability !== 'mergeable') {
            return ['merged' => false, 'reason' => 'conflicts'];
        }

        if (! $this->checksHavePassed($prMeta->checkStatuses)) {
            return ['merged' => false, 'reason' => 'checks_pending'];
        }

        $forge->mergePullRequest($owner, $repo, $prNumber);

        return ['merged' => true, 'pr_number' => $prNumber];
    }

    /**
     * @param  array<int, array{name: string, conclusion: string|null, status: string|null}>  $checkStatuses
     */
    private function checksHavePassed(array $checkStatuses): bool
    {
        if ($checkStatuses === []) {
            return false;
        }

        foreach ($checkStatuses as $checkStatus) {
            if (($checkStatus['status'] ?? null) !== 'completed') {
                return false;
            }

            if (($checkStatus['conclusion'] ?? null) !== 'success') {
                return false;
            }
        }

        return true;
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
