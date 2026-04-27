<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Console;

use Carbon\CarbonImmutable;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

class PruneMetricsCommand extends Command
{
    private const TABLE = 'mcp_tool_metrics';

    protected $signature = 'mcp:prune-metrics
        {--days=30 : Remove metric rows older than this many days}';

    protected $description = 'Prune old MCP tool metrics';

    public function handle(): int
    {
        $days = $this->days();
        if ($days === null) {
            return self::FAILURE;
        }

        if (! Schema::hasTable(self::TABLE)) {
            $this->error('The mcp_tool_metrics table is required for metric pruning.');

            return self::FAILURE;
        }

        $cutoffDate = CarbonImmutable::now()->subDays($days)->startOfDay()->toDateString();
        $staleQuery = DB::table(self::TABLE)->where('date', '<', $cutoffDate);
        $staleCount = (clone $staleQuery)->count();

        if ($staleCount === 0) {
            $this->info(sprintf(
                'No MCP metric records older than %d day(s) were found.',
                $days,
            ));

            return self::SUCCESS;
        }

        $summary = (clone $staleQuery)
            ->selectRaw('tool_id, workspace_id, COUNT(*) as metric_rows, COALESCE(SUM(call_count), 0) as total_calls')
            ->groupBy('tool_id', 'workspace_id')
            ->get();
        $deleted = $staleQuery->delete();
        $bucketCount = $summary->count();
        $totalCalls = (int) $summary->sum('total_calls');

        $this->info(sprintf(
            'Pruned %d MCP metric record(s) older than %d day(s) across %d bucket(s) covering %d call(s).',
            $deleted,
            $days,
            $bucketCount,
            $totalCalls,
        ));

        return self::SUCCESS;
    }

    private function days(): int|null
    {
        $days = filter_var($this->option('days'), FILTER_VALIDATE_INT);

        if ($days === false || $days < 1) {
            $this->error('--days must be a positive integer.');

            return null;
        }

        return $days;
    }
}
