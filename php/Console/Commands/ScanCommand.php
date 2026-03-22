<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Actions\Forge\CreatePlanFromIssue;
use Core\Mod\Agentic\Actions\Forge\ReportToIssue;
use Core\Mod\Agentic\Actions\Forge\ScanForWork;
use Illuminate\Console\Command;

class ScanCommand extends Command
{
    protected $signature = 'agentic:scan
        {--workspace=1 : Workspace ID}
        {--repos=* : Repos to scan (owner/name format)}
        {--dry-run : Show what would be created without acting}';

    protected $description = 'Scan Forgejo repositories for actionable work from epic issues';

    public function handle(): int
    {
        $workspaceId = (int) $this->option('workspace');
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
        $totalItems = 0;

        foreach ($repos as $repoSpec) {
            $parts = explode('/', $repoSpec, 2);

            if (count($parts) !== 2) {
                $this->error("Invalid repo format: {$repoSpec} (expected owner/name)");

                continue;
            }

            [$owner, $repo] = $parts;

            $this->info("Scanning {$owner}/{$repo}...");

            $workItems = ScanForWork::run($owner, $repo);

            if (empty($workItems)) {
                $this->line("  No actionable work found.");

                continue;
            }

            foreach ($workItems as $item) {
                $totalItems++;
                $issueNumber = $item['issue_number'];
                $title = $item['issue_title'];

                if ($isDryRun) {
                    $this->line("  DRY RUN: Would create plan for #{$issueNumber} — {$title}");

                    continue;
                }

                $plan = CreatePlanFromIssue::run($item, $workspaceId);

                ReportToIssue::run(
                    $owner,
                    $repo,
                    $issueNumber,
                    "Plan created: **{$plan->title}** (#{$plan->id})"
                );

                $this->line("  Created plan #{$plan->id} for issue #{$issueNumber}: {$title}");
            }
        }

        $action = $isDryRun ? 'found' : 'processed';
        $this->info("Scan complete: {$totalItems} work item(s) {$action}.");

        return self::SUCCESS;
    }
}
