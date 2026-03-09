<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Actions\Forge\AssignAgent;
use Core\Mod\Agentic\Actions\Forge\ReportToIssue;
use Core\Mod\Agentic\Models\AgentPlan;
use Illuminate\Console\Command;

class DispatchCommand extends Command
{
    protected $signature = 'agentic:dispatch
        {--workspace=1 : Workspace ID}
        {--agent-type=opus : Default agent type}
        {--dry-run : Show what would be dispatched}';

    protected $description = 'Dispatch agents to draft plans sourced from Forgejo';

    public function handle(): int
    {
        $workspaceId = (int) $this->option('workspace');
        $defaultAgentType = (string) $this->option('agent-type');
        $isDryRun = (bool) $this->option('dry-run');

        $plans = AgentPlan::where('status', AgentPlan::STATUS_DRAFT)
            ->whereJsonContains('metadata->source', 'forgejo')
            ->whereDoesntHave('sessions')
            ->get();

        if ($plans->isEmpty()) {
            $this->info('No draft Forgejo plans awaiting dispatch.');

            return self::SUCCESS;
        }

        $dispatched = 0;

        foreach ($plans as $plan) {
            $assignee = $plan->metadata['assignee'] ?? $defaultAgentType;
            $issueNumber = $plan->metadata['issue_number'] ?? null;
            $owner = $plan->metadata['repo_owner'] ?? null;
            $repo = $plan->metadata['repo_name'] ?? null;

            if ($isDryRun) {
                $this->line("DRY RUN: Would dispatch '{$assignee}' to plan #{$plan->id} — {$plan->title}");
                $dispatched++;

                continue;
            }

            $session = AssignAgent::run($plan, $assignee, $workspaceId);

            if ($issueNumber !== null && $owner !== null && $repo !== null) {
                ReportToIssue::run(
                    (string) $owner,
                    (string) $repo,
                    (int) $issueNumber,
                    "Agent **{$assignee}** dispatched. Session: #{$session->id}"
                );
            }

            $this->line("Dispatched '{$assignee}' to plan #{$plan->id}: {$plan->title} (session #{$session->id})");
            $dispatched++;
        }

        $action = $isDryRun ? 'would be dispatched' : 'dispatched';
        $this->info("{$dispatched} plan(s) {$action}.");

        return self::SUCCESS;
    }
}
