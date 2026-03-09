<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\AgentPlan;
use Illuminate\Console\Command;

class PlanRetentionCommand extends Command
{
    protected $signature = 'agentic:plan-cleanup
        {--dry-run : Preview deletions without making changes}
        {--days= : Override retention period (overrides agentic.plan_retention_days config)}';

    protected $description = 'Permanently delete archived plans past the retention period';

    public function handle(): int
    {
        $days = (int) ($this->option('days') ?? config('agentic.plan_retention_days', 90));

        if ($days <= 0) {
            $this->info('Retention cleanup is disabled (plan_retention_days is 0).');

            return self::SUCCESS;
        }

        $cutoff = now()->subDays($days);

        $query = AgentPlan::where('status', AgentPlan::STATUS_ARCHIVED)
            ->whereNotNull('archived_at')
            ->where('archived_at', '<', $cutoff);

        $count = $query->count();

        if ($count === 0) {
            $this->info('No archived plans found past the retention period.');

            return self::SUCCESS;
        }

        if ($this->option('dry-run')) {
            $this->info("DRY RUN: {$count} archived plan(s) would be permanently deleted (archived before {$cutoff->toDateString()}).");

            return self::SUCCESS;
        }

        $deleted = 0;

        $query->chunkById(100, function ($plans) use (&$deleted): void {
            foreach ($plans as $plan) {
                $plan->forceDelete();
                $deleted++;
            }
        });

        $this->info("Permanently deleted {$deleted} archived plan(s) archived before {$cutoff->toDateString()}.");

        return self::SUCCESS;
    }
}
