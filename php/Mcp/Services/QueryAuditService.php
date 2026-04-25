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

final class QueryAuditService
{
    private const TABLE = 'mcp_audit_entries';

    public function isSafe(string $query): bool
    {
        return preg_match(
            '/\b(drop|delete|truncate|alter|create|insert|update)\b|(?:exec|system|passthru)\s*\(/i',
            $query,
        ) !== 1;
    }

    public function exceedsLimit(array $result, int $limitBytes = 1000000): bool
    {
        return strlen((string) json_encode($result, JSON_INVALID_UTF8_SUBSTITUTE)) > $limitBytes;
    }

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
        $entries = McpAuditEntry::query()->orderBy('created_at')->get();
        $aggregates = [];

        foreach ($resolvedPeriods as $period) {
            $resolvedPeriod = $this->resolvePeriod((string) $period);

            $aggregates[$resolvedPeriod] = $entries->groupBy(
                fn (McpAuditEntry $entry): string => $this->bucketFor(
                    $entry->created_at instanceof CarbonInterface
                        ? CarbonImmutable::instance($entry->created_at)
                        : CarbonImmutable::parse((string) ($entry->created_at ?? 'now')),
                    $resolvedPeriod,
                ),
            )->map(
                static function (Collection $group, string $bucket): array {
                    return [
                        'bucket' => $bucket,
                        'total' => $group->count(),
                        'safe' => $group->where('is_safe', true)->count(),
                        'unsafe' => $group->where('is_safe', false)->count(),
                        'average_duration_ms' => (int) round((float) ($group->avg('duration_ms') ?? 0)),
                        'result_count' => (int) $group->sum('result_count'),
                    ];
                },
            )->values()->all();
        }

        return $aggregates;
    }

    private function ensureTableExists(): void
    {
        if (! Schema::hasTable(self::TABLE)) {
            throw new RuntimeException('The mcp_audit_entries table is required for QueryAuditService.');
        }
    }

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

    private function bucketFor(CarbonImmutable $timestamp, string $period): string
    {
        return match ($period) {
            'minute' => $timestamp->format('Y-m-d H:i'),
            'hour' => $timestamp->format('Y-m-d H:00'),
            'day' => $timestamp->format('Y-m-d'),
        };
    }
}

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
