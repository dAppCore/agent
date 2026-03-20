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
 * Post a progress comment on a Forgejo issue.
 *
 * Wraps ForgejoService::createComment() for use as a
 * standalone action within the orchestration pipeline.
 *
 * Usage:
 *   ReportToIssue::run('core', 'app', 5, 'Phase 1 complete.');
 */
class ReportToIssue
{
    use Action;

    public function handle(string $owner, string $repo, int $issueNumber, string $message): void
    {
        app(ForgejoService::class)->createComment($owner, $repo, $issueNumber, $message);
    }
}
