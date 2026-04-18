<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Actions\Forge\ManagePullRequest;
use Core\Mod\Agentic\Services\ForgejoService;
use Illuminate\Console\Command;

class PrManageCommand extends Command
{
    protected $signature = 'agentic:pr-manage
        {--repos=* : Repos to manage (owner/name format)}
        {--dry-run : Show what would be merged}';

    protected $description = 'Review and merge ready pull requests on Forgejo repositories';

    public function handle(): int
    {
        $repos = $this->option('repos');

        if (empty($repos)) {
            $repos = config('agentic.scan_repos', []);
        }

        $repos = array_filter($repos);

        if (empty($repos)) {
            $this->warn('No repositories configured. Pass --repos or set AGENTIC_SCAN_REPOS.');

            return self::SUCCESS;
        }

        $isDryRun = (bool) $this->option('dry-run');
        $forge = app(ForgejoService::class);
        $totalProcessed = 0;

        foreach ($repos as $repoSpec) {
            $parts = explode('/', $repoSpec, 2);

            if (count($parts) !== 2) {
                $this->error("Invalid repo format: {$repoSpec} (expected owner/name)");

                continue;
            }

            [$owner, $repo] = $parts;

            $this->info("Checking PRs for {$owner}/{$repo}...");

            $pullRequests = $forge->listPullRequests($owner, $repo, 'open');

            if (empty($pullRequests)) {
                $this->line("  No open PRs.");

                continue;
            }

            foreach ($pullRequests as $pr) {
                $prNumber = (int) $pr['number'];
                $prTitle = (string) ($pr['title'] ?? '');
                $totalProcessed++;

                if ($isDryRun) {
                    $this->line("  DRY RUN: Would evaluate PR #{$prNumber} — {$prTitle}");

                    continue;
                }

                $result = ManagePullRequest::run($owner, $repo, $prNumber);

                if ($result['merged']) {
                    $this->line("  Merged PR #{$prNumber}: {$prTitle}");
                } else {
                    $reason = $result['reason'] ?? 'unknown';
                    $this->line("  Skipped PR #{$prNumber}: {$prTitle} ({$reason})");
                }
            }
        }

        $action = $isDryRun ? 'found' : 'processed';
        $this->info("PR management complete: {$totalProcessed} PR(s) {$action}.");

        return self::SUCCESS;
    }
}
