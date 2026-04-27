<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Services;

use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;
use Core\Mod\Agentic\Mcp\Data\AuditEntry;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Schema;
use InvalidArgumentException;
use RuntimeException;

/**
 * Audit MCP query usage and expose filtered activity summaries.
 *
 * @example
 * $service = new QueryAuditService();
 * $entry = $service->log('select * from memories', ['workspace_id' => '42']);
 */
final class QueryAuditService
{
    private const TABLE = 'mcp_audit_entries';

    /**
     * Decide whether a query looks safe enough to execute.
     *
     * @example
     * $safe = $service->isSafe('select * from memories');
     */
    public function isSafe(string $query): bool
    {
        $trimmedQuery = ltrim($query);
        $startsWithWriteStatement = preg_match(
            '/^(?:--[^\n]*\n\s*)*(?:drop|delete|truncate|alter|create|insert|update)\b/i',
            $trimmedQuery,
        ) === 1;
        $callsDangerousFunction = preg_match('/(?:exec|system|passthru)\s*\(/i', $query) === 1;

        return ! $startsWithWriteStatement && ! $callsDangerousFunction;
    }

    /**
     * Decide whether an encoded result payload exceeds the audit size limit.
     *
     * @example
     * $tooLarge = $service->exceedsLimit(['rows' => range(1, 5000)], 1024);
     */
    public function exceedsLimit(array $result, int $limitBytes = 1000000): bool
    {
        return strlen((string) json_encode($result, JSON_INVALID_UTF8_SUBSTITUTE)) > $limitBytes;
    }

    /**
     * Persist one audit entry for a query execution.
     *
     * @example
     * $entry = $service->log('select * from memories', ['workspace_id' => '42', 'tool_name' => 'brain_list']);
     */
    public function log(string $query, array $context = []): AuditEntry
    {
        $this->ensureTableExists();

        $recordedAt = $this->resolveRecordedAt($context['recorded_at'] ?? null);

        $entry = McpAuditEntry::query()->create([
            'workspace_id' => isset($context['workspace_id']) ? (string) $context['workspace_id'] : null,
            'tool_name' => isset($context['tool_name']) ? (string) $context['tool_name'] : null,
            'query_text' => $query,
            'query_hash' => hash('sha256', $query),
            'is_safe' => $this->isSafe($query),
            'result_count' => isset($context['result_count']) ? (int) $context['result_count'] : null,
            'duration_ms' => isset($context['duration_ms']) ? (int) $context['duration_ms'] : null,
            'metadata' => (array) ($context['metadata'] ?? []),
            'created_at' => $recordedAt,
            'updated_at' => $recordedAt,
        ]);

        return AuditEntry::fromModel($entry);
    }

    /**
     * @return Collection<int, AuditEntry>
     */
    public function query(array $filters = []): Collection
    {
        $this->ensureTableExists();

        $limit = (int) ($filters['limit'] ?? 100);
        if ($limit < 1) {
            throw new InvalidArgumentException('Query filters require limit to be at least 1.');
        }

        $builder = McpAuditEntry::query()->orderByDesc('created_at');

        if (array_key_exists('workspace_id', $filters)) {
            $builder->where('workspace_id', (string) $filters['workspace_id']);
        }

        if (array_key_exists('workspace', $filters)) {
            $builder->where('workspace_id', (string) $filters['workspace']);
        }

        if (array_key_exists('tool_name', $filters)) {
            $builder->where('tool_name', (string) $filters['tool_name']);
        }

        if (array_key_exists('tool', $filters)) {
            $builder->where('tool_name', (string) $filters['tool']);
        }

        if (array_key_exists('safe', $filters)) {
            $builder->where('is_safe', (bool) $filters['safe']);
        }

        if (array_key_exists('is_safe', $filters)) {
            $builder->where('is_safe', (bool) $filters['is_safe']);
        }

        if (array_key_exists('search', $filters)) {
            $builder->where('query_text', 'like', '%'.(string) $filters['search'].'%');
        }

        if (array_key_exists('from', $filters)) {
            $builder->where('created_at', '>=', $this->resolveRecordedAt($filters['from']));
        }

        if (array_key_exists('until', $filters)) {
            $builder->where('created_at', '<=', $this->resolveRecordedAt($filters['until']));
        }

        return $builder->limit($limit)->get()->map(
            static fn (McpAuditEntry $entry): AuditEntry => AuditEntry::fromModel($entry),
        );
    }

    /**
     * @return array<string, array<int, array<string, int|string>>>
     */
    public function aggregate(array $periods = ['day']): array
    {
        $this->ensureTableExists();

        $resolvedPeriods = $periods === [] ? ['day'] : array_values(array_unique($periods));
        $aggregates = [];

        foreach ($resolvedPeriods as $period) {
            $resolvedPeriod = $this->resolvePeriod((string) $period);
            $aggregates[$resolvedPeriod] = [];
        }

        McpAuditEntry::query()
            ->orderBy('id')
            ->chunkById(250, function (Collection $entries) use (&$aggregates, $resolvedPeriods): void {
                foreach ($entries as $entry) {
                    $timestamp = $this->entryTimestamp($entry);

                    foreach ($resolvedPeriods as $resolvedPeriod) {
                        $bucket = $this->bucketFor($timestamp, $resolvedPeriod);

                        if (! isset($aggregates[$resolvedPeriod][$bucket])) {
                            $aggregates[$resolvedPeriod][$bucket] = [
                                'bucket' => $bucket,
                                'total' => 0,
                                'safe' => 0,
                                'unsafe' => 0,
                                'duration_total' => 0,
                                'result_count' => 0,
                            ];
                        }

                        $aggregates[$resolvedPeriod][$bucket]['total']++;
                        $aggregates[$resolvedPeriod][$bucket][$entry->is_safe ? 'safe' : 'unsafe']++;
                        $aggregates[$resolvedPeriod][$bucket]['duration_total'] += (int) ($entry->duration_ms ?? 0);
                        $aggregates[$resolvedPeriod][$bucket]['result_count'] += (int) ($entry->result_count ?? 0);
                    }
                }
            });

        foreach ($aggregates as $period => $buckets) {
            ksort($buckets);

            $aggregates[$period] = array_values(array_map(
                static function (array $bucket): array {
                    $total = max((int) $bucket['total'], 1);

                    return [
                        'bucket' => (string) $bucket['bucket'],
                        'total' => (int) $bucket['total'],
                        'safe' => (int) $bucket['safe'],
                        'unsafe' => (int) $bucket['unsafe'],
                        'average_duration_ms' => (int) round(((int) $bucket['duration_total']) / $total),
                        'result_count' => (int) $bucket['result_count'],
                    ];
                },
                $buckets,
            ));
        }

        return $aggregates;
    }

    /**
     * Assert that the audit table exists before reading or writing entries.
     *
     * @example
     * $this->ensureTableExists();
     */
    private function ensureTableExists(): void
    {
        if (! Schema::hasTable(self::TABLE)) {
            throw new RuntimeException('The mcp_audit_entries table is required for QueryAuditService.');
        }
    }

    /**
     * Normalise a recorded-at value into an immutable timestamp.
     *
     * @example
     * $recordedAt = $this->resolveRecordedAt('2026-04-27T10:15:00Z');
     */
    private function resolveRecordedAt(mixed $value): CarbonImmutable
    {
        if ($value instanceof CarbonImmutable) {
            return $value;
        }

        if ($value instanceof CarbonInterface) {
            return CarbonImmutable::instance($value);
        }

        if ($value === null || $value === '') {
            return CarbonImmutable::now();
        }

        return CarbonImmutable::parse((string) $value);
    }

    /**
     * Validate and return a supported aggregation period.
     *
     * @example
     * $period = $this->resolvePeriod('hour');
     */
    private function resolvePeriod(string $period): string
    {
        if (! in_array($period, ['minute', 'hour', 'day'], true)) {
            throw new InvalidArgumentException(sprintf(
                'Unsupported aggregation period [%s].',
                $period,
            ));
        }

        return $period;
    }

    /**
     * Format one aggregation bucket key for the requested period.
     *
     * @example
     * $bucket = $this->bucketFor(CarbonImmutable::parse('2026-04-27 10:15:00'), 'hour');
     */
    private function bucketFor(CarbonImmutable $timestamp, string $period): string
    {
        return match ($period) {
            'minute' => $timestamp->format('Y-m-d H:i'),
            'hour' => $timestamp->format('Y-m-d H:00'),
            'day' => $timestamp->format('Y-m-d'),
        };
    }

    /**
     * Resolve an audit entry model timestamp into an immutable Carbon value.
     *
     * @example
     * $timestamp = $this->entryTimestamp($entry);
     */
    private function entryTimestamp(McpAuditEntry $entry): CarbonImmutable
    {
        if ($entry->created_at instanceof CarbonInterface) {
            return CarbonImmutable::instance($entry->created_at);
        }

        return CarbonImmutable::parse((string) ($entry->created_at ?? 'now'));
    }
}

/**
 * Persisted Eloquent model for one MCP audit log entry.
 *
 * @example
 * $entry = McpAuditEntry::query()->first();
 */
class McpAuditEntry extends Model
{
    protected $table = 'mcp_audit_entries';

    protected $fillable = [
        'workspace_id',
        'tool_name',
        'query_text',
        'query_hash',
        'is_safe',
        'result_count',
        'duration_ms',
        'metadata',
        'created_at',
        'updated_at',
    ];

    protected $casts = [
        'is_safe' => 'bool',
        'result_count' => 'int',
        'duration_ms' => 'int',
        'metadata' => 'array',
        'created_at' => 'datetime',
        'updated_at' => 'datetime',
    ];
}
