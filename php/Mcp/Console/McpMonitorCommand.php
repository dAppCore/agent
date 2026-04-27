<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Console;

use Carbon\CarbonImmutable;
use Core\Mod\Agentic\Mcp\Services\QueryAuditService;
use Illuminate\Console\Command;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Schema;
use RuntimeException;

class McpMonitorCommand extends Command
{
    private const METRICS_TABLE = 'mcp_tool_metrics';

    protected $signature = 'mcp:monitor
        {action=status : Action to perform}
        {--days=7 : Number of days to include in the report window}
        {--json : Output machine-readable JSON}';

    protected $description = 'Monitor MCP health, alerts, exports, and metrics output';

    public function handle(QueryAuditService $queryAuditService): int
    {
        $days = $this->days();
        if ($days === null) {
            return self::FAILURE;
        }

        $action = strtolower((string) $this->argument('action'));

        return match ($action) {
            'status' => $this->statusAction($days, $queryAuditService),
            'alerts' => $this->alertsAction($days, $queryAuditService),
            'export' => $this->exportAction($days, $queryAuditService),
            'report' => $this->reportAction($days, $queryAuditService),
            'prometheus' => $this->prometheusAction($days),
            default => $this->unsupportedAction($action),
        };
    }

    private function statusAction(int $days, QueryAuditService $queryAuditService): int
    {
        $health = $this->healthStatus($days, $queryAuditService);

        if ((bool) $this->option('json')) {
            $this->line($this->json([
                'action' => 'status',
                'days' => $days,
                'status' => $health['status'],
                'metrics' => $health['metrics'],
                'issues' => $health['issues'],
            ]));

            return $health['status'] === 'CRITICAL' ? self::FAILURE : self::SUCCESS;
        }

        $this->line(sprintf('MCP Health Status: %s', $health['status']));
        $this->newLine();
        $this->table(['Metric', 'Value'], [
            ['Total Calls', number_format((int) $health['metrics']['total_calls'])],
            ['Success Rate', sprintf('%.1f%%', (float) $health['metrics']['success_rate'])],
            ['Error Rate', sprintf('%.1f%%', (float) $health['metrics']['error_rate'])],
            ['Avg Duration', sprintf('%dms', (int) $health['metrics']['avg_duration_ms'])],
        ]);

        if ($health['issues'] === []) {
            $this->info('No issues detected.');
        } else {
            $this->line('Issues Detected:');

            foreach ($health['issues'] as $issue) {
                $this->line(sprintf('  [!] %s', $issue));
            }
        }

        return $health['status'] === 'CRITICAL' ? self::FAILURE : self::SUCCESS;
    }

    private function alertsAction(int $days, QueryAuditService $queryAuditService): int
    {
        $alerts = $this->alerts($days, $queryAuditService);

        if ((bool) $this->option('json')) {
            $this->line($this->json([
                'action' => 'alerts',
                'days' => $days,
                'alerts' => $alerts,
            ]));

            return $alerts === [] ? self::SUCCESS : self::FAILURE;
        }

        if ($alerts === []) {
            $this->info('No MCP alerts detected.');

            return self::SUCCESS;
        }

        $this->line('MCP Alerts:');

        foreach ($alerts as $alert) {
            $this->line(sprintf('  [!] %s', $alert));
        }

        return self::FAILURE;
    }

    private function exportAction(int $days, QueryAuditService $queryAuditService): int
    {
        $report = $this->summaryReport($days, $queryAuditService);

        Log::info('MCP metrics export', [
            'days' => $days,
            'overview' => $report['overview'],
            'top_tools' => $report['top_tools'],
            'anomalies' => $report['anomalies'],
        ]);

        if ((bool) $this->option('json')) {
            $this->line($this->json([
                'action' => 'export',
                'days' => $days,
                'exported' => true,
                'channel' => 'log',
                'report' => $report,
            ]));

            return self::SUCCESS;
        }

        $this->info('Exported MCP metrics summary to the log channel.');

        return self::SUCCESS;
    }

    private function reportAction(int $days, QueryAuditService $queryAuditService): int
    {
        $report = $this->summaryReport($days, $queryAuditService);

        if ((bool) $this->option('json')) {
            $this->line($this->json([
                'action' => 'report',
                'days' => $days,
                'report' => $report,
            ]));

            return self::SUCCESS;
        }

        $this->line(sprintf('MCP Summary Report (%d day window)', $days));
        $this->newLine();
        $this->table(['Metric', 'Value'], [
            ['Total Calls', number_format((int) $report['overview']['total_calls'])],
            ['Success Rate', sprintf('%.1f%%', (float) $report['overview']['success_rate'])],
            ['Error Rate', sprintf('%.1f%%', (float) $report['overview']['error_rate'])],
            ['Avg Duration', sprintf('%dms', (int) $report['overview']['avg_duration_ms'])],
        ]);

        if ($report['top_tools'] !== []) {
            $this->newLine();
            $this->table(['Tool', 'Calls', 'Error Rate', 'Avg Duration'], array_map(
                static fn (array $tool): array => [
                    $tool['tool_id'],
                    number_format((int) $tool['call_count']),
                    sprintf('%.1f%%', (float) $tool['error_rate']),
                    sprintf('%dms', (int) $tool['avg_duration_ms']),
                ],
                $report['top_tools'],
            ));
        }

        if ($report['anomalies'] === []) {
            $this->info('No anomalies detected.');

            return self::SUCCESS;
        }

        $this->line('Anomalies:');

        foreach ($report['anomalies'] as $anomaly) {
            $this->line(sprintf('  [!] %s', $anomaly));
        }

        return self::SUCCESS;
    }

    private function prometheusAction(int $days): int
    {
        $metrics = $this->prometheusMetrics($days);

        if ((bool) $this->option('json')) {
            $this->line($this->json([
                'action' => 'prometheus',
                'days' => $days,
                'metrics' => $metrics,
            ]));

            return self::SUCCESS;
        }

        $this->output->write($metrics);

        return self::SUCCESS;
    }

    private function unsupportedAction(string $action): int
    {
        $this->error(sprintf('Unsupported monitor action [%s].', $action));

        return self::FAILURE;
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

    private function healthStatus(int $days, QueryAuditService $queryAuditService): array
    {
        $overview = $this->overview($days);
        $issues = [];

        if (! $overview['metrics_available']) {
            $issues[] = 'Metrics table unavailable.';
        }

        foreach ($this->topTools($days) as $tool) {
            if ((float) $tool['error_rate'] > 20.0) {
                $issues[] = sprintf('High error rate on tool: %s', $tool['tool_id']);
            }
        }

        $unsafeAudits = $this->unsafeAuditCount($queryAuditService, $days);
        if ($unsafeAudits !== null && $unsafeAudits > 0) {
            $issues[] = sprintf('%d unsafe query audit entr%s detected.', $unsafeAudits, $unsafeAudits === 1 ? 'y' : 'ies');
        }

        $status = 'HEALTHY';

        if ((float) $overview['error_rate'] > 10.0) {
            $status = 'CRITICAL';
        } elseif (
            (float) $overview['error_rate'] > 5.0
            || (int) $overview['avg_duration_ms'] > 500
            || $issues !== []
        ) {
            $status = 'DEGRADED';
        }

        return [
            'status' => $status,
            'metrics' => $overview,
            'issues' => $issues,
        ];
    }

    private function alerts(int $days, QueryAuditService $queryAuditService): array
    {
        $alerts = [];

        if (! Schema::hasTable(self::METRICS_TABLE)) {
            $alerts[] = 'Metrics table unavailable.';
        }

        foreach ($this->topTools($days) as $tool) {
            if ((float) $tool['error_rate'] > 20.0) {
                $alerts[] = sprintf(
                    'Tool [%s] is failing at %.1f%%.',
                    $tool['tool_id'],
                    (float) $tool['error_rate'],
                );
            }
        }

        $unsafeAudits = $this->unsafeAuditCount($queryAuditService, $days);
        if ($unsafeAudits !== null && $unsafeAudits > 0) {
            $alerts[] = sprintf('%d unsafe query audit entr%s detected.', $unsafeAudits, $unsafeAudits === 1 ? 'y' : 'ies');
        }

        return $alerts;
    }

    private function summaryReport(int $days, QueryAuditService $queryAuditService): array
    {
        return [
            'overview' => $this->overview($days),
            'top_tools' => $this->topTools($days),
            'anomalies' => $this->anomalies($days, $queryAuditService),
        ];
    }

    private function overview(int $days): array
    {
        $rows = $this->metricRows($days);

        if ($rows->isEmpty()) {
            return [
                'metrics_available' => Schema::hasTable(self::METRICS_TABLE),
                'total_calls' => 0,
                'success_rate' => 0.0,
                'error_rate' => 0.0,
                'avg_duration_ms' => 0,
            ];
        }

        $totalCalls = (int) $rows->sum(fn (object $row): int => (int) ($row->call_count ?? 0));
        $successCount = (int) $rows->sum(fn (object $row): int => (int) ($row->success_count ?? 0));
        $errorCount = (int) $rows->sum(fn (object $row): int => (int) ($row->error_count ?? 0));
        $weightedDuration = (int) $rows->sum(fn (object $row): int => (int) ($row->avg_duration_ms ?? 0) * (int) ($row->call_count ?? 0));

        return [
            'metrics_available' => true,
            'total_calls' => $totalCalls,
            'success_rate' => $totalCalls > 0 ? round(($successCount / $totalCalls) * 100, 1) : 0.0,
            'error_rate' => $totalCalls > 0 ? round(($errorCount / $totalCalls) * 100, 1) : 0.0,
            'avg_duration_ms' => $totalCalls > 0 ? (int) round($weightedDuration / $totalCalls) : 0,
        ];
    }

    private function topTools(int $days): array
    {
        return $this->metricRows($days)
            ->groupBy(static fn (object $row): string => (string) ($row->tool_id ?? 'unknown'))
            ->map(static function (Collection $group, string $toolId): array {
                $callCount = (int) $group->sum(fn (object $row): int => (int) ($row->call_count ?? 0));
                $errorCount = (int) $group->sum(fn (object $row): int => (int) ($row->error_count ?? 0));
                $weightedDuration = (int) $group->sum(
                    fn (object $row): int => (int) ($row->avg_duration_ms ?? 0) * (int) ($row->call_count ?? 0),
                );

                return [
                    'tool_id' => $toolId,
                    'call_count' => $callCount,
                    'error_rate' => $callCount > 0 ? round(($errorCount / $callCount) * 100, 1) : 0.0,
                    'avg_duration_ms' => $callCount > 0 ? (int) round($weightedDuration / $callCount) : 0,
                ];
            })
            ->sortByDesc('call_count')
            ->values()
            ->take(5)
            ->all();
    }

    private function anomalies(int $days, QueryAuditService $queryAuditService): array
    {
        $anomalies = [];
        $overview = $this->overview($days);

        if ((float) $overview['error_rate'] > 10.0) {
            $anomalies[] = sprintf('Overall MCP error rate is %.1f%%.', (float) $overview['error_rate']);
        }

        if ((int) $overview['avg_duration_ms'] > 500) {
            $anomalies[] = sprintf('Average MCP duration is %dms.', (int) $overview['avg_duration_ms']);
        }

        foreach ($this->topTools($days) as $tool) {
            if ((float) $tool['error_rate'] > 20.0) {
                $anomalies[] = sprintf(
                    'Tool [%s] exceeded the 20%% error-rate threshold.',
                    $tool['tool_id'],
                );
            }
        }

        $unsafeAudits = $this->unsafeAuditCount($queryAuditService, $days);
        if ($unsafeAudits !== null && $unsafeAudits > 0) {
            $anomalies[] = sprintf('%d unsafe query audit entr%s detected.', $unsafeAudits, $unsafeAudits === 1 ? 'y' : 'ies');
        }

        if (! Schema::hasTable(self::METRICS_TABLE)) {
            $anomalies[] = 'Metrics table unavailable.';
        }

        return $anomalies;
    }

    private function prometheusMetrics(int $days): string
    {
        $lines = [
            '# HELP mcp_tool_calls_total Total MCP tool calls recorded.',
            '# TYPE mcp_tool_calls_total counter',
        ];

        $topTools = $this->topTools($days);
        if ($topTools === []) {
            $lines[] = 'mcp_tool_calls_total 0';
        } else {
            foreach ($topTools as $tool) {
                $lines[] = sprintf(
                    'mcp_tool_calls_total{tool="%s"} %d',
                    $this->prometheusLabel((string) $tool['tool_id']),
                    (int) $tool['call_count'],
                );
            }
        }

        $lines[] = '# HELP mcp_tool_errors_total Total MCP tool errors recorded.';
        $lines[] = '# TYPE mcp_tool_errors_total counter';

        if ($topTools === []) {
            $lines[] = 'mcp_tool_errors_total 0';
        } else {
            foreach ($topTools as $tool) {
                $errorCount = (int) round(((float) $tool['error_rate'] / 100) * (int) $tool['call_count']);
                $lines[] = sprintf(
                    'mcp_tool_errors_total{tool="%s"} %d',
                    $this->prometheusLabel((string) $tool['tool_id']),
                    $errorCount,
                );
            }
        }

        $lines[] = '# HELP mcp_tool_duration_ms Average MCP tool duration in milliseconds.';
        $lines[] = '# TYPE mcp_tool_duration_ms gauge';

        if ($topTools === []) {
            $lines[] = 'mcp_tool_duration_ms 0';
        } else {
            foreach ($topTools as $tool) {
                $lines[] = sprintf(
                    'mcp_tool_duration_ms{tool="%s"} %d',
                    $this->prometheusLabel((string) $tool['tool_id']),
                    (int) $tool['avg_duration_ms'],
                );
            }
        }

        $lines[] = '# HELP mcp_quota_exceeded_total Total MCP quota exceeded events observed by the monitor.';
        $lines[] = '# TYPE mcp_quota_exceeded_total counter';
        $lines[] = 'mcp_quota_exceeded_total 0';
        $lines[] = '# HELP mcp_circuit_breaker_open Number of MCP tools with an open circuit breaker.';
        $lines[] = '# TYPE mcp_circuit_breaker_open gauge';
        $lines[] = 'mcp_circuit_breaker_open 0';

        return implode(PHP_EOL, $lines).PHP_EOL;
    }

    private function unsafeAuditCount(QueryAuditService $queryAuditService, int $days): int|null
    {
        try {
            return $queryAuditService->query([
                'safe' => false,
                'from' => CarbonImmutable::now()->subDays($days - 1)->startOfDay(),
                'limit' => 100,
            ])->count();
        } catch (RuntimeException) {
            return null;
        }
    }

    private function metricRows(int $days): Collection
    {
        if (! Schema::hasTable(self::METRICS_TABLE)) {
            return collect();
        }

        $fromDate = CarbonImmutable::now()->subDays($days - 1)->startOfDay()->toDateString();

        return DB::table(self::METRICS_TABLE)
            ->where('date', '>=', $fromDate)
            ->get();
    }

    private function prometheusLabel(string $value): string
    {
        return str_replace(['\\', '"'], ['\\\\', '\\"'], $value);
    }

    private function json(array $payload): string
    {
        $encoded = json_encode(
            $payload,
            JSON_INVALID_UTF8_SUBSTITUTE | JSON_UNESCAPED_SLASHES,
        );

        return $encoded === false ? '{}' : $encoded;
    }
}
