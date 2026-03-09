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
 * Evaluate and merge a Forgejo pull request when ready.
 *
 * Checks the PR state, mergeability, and CI status before
 * attempting the merge. Returns a result array describing
 * the outcome.
 *
 * Usage:
 *   $result = ManagePullRequest::run('core', 'app', 10);
 */
class ManagePullRequest
{
    use Action;

    /**
     * @return array{merged: bool, pr_number?: int, reason?: string}
     */
    public function handle(string $owner, string $repo, int $prNumber): array
    {
        $forge = app(ForgejoService::class);

        $pr = $forge->getPullRequest($owner, $repo, $prNumber);

        if (($pr['state'] ?? '') !== 'open') {
            return ['merged' => false, 'reason' => 'not_open'];
        }

        if (empty($pr['mergeable'])) {
            return ['merged' => false, 'reason' => 'conflicts'];
        }

        $headSha = $pr['head']['sha'] ?? '';
        $status = $forge->getCombinedStatus($owner, $repo, $headSha);

        if (($status['state'] ?? '') !== 'success') {
            return ['merged' => false, 'reason' => 'checks_pending'];
        }

        $forge->mergePullRequest($owner, $repo, $prNumber);

        return ['merged' => true, 'pr_number' => $prNumber];
    }
}
