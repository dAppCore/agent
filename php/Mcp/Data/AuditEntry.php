<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Data;

use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;

final readonly class AuditEntry
{
    public function __construct(
        public ?int $id,
        public ?string $workspaceId,
        public ?string $toolName,
        public string $query,
        public string $queryHash,
        public bool $isSafe,
        public ?int $resultCount,
        public ?int $durationMs,
        public array $metadata,
        public CarbonImmutable $createdAt,
    ) {}

    public static function fromModel(object $entry): self
    {
        $createdAt = $entry->created_at ?? null;

        if ($createdAt instanceof CarbonImmutable) {
            $immutable = $createdAt;
        } elseif ($createdAt instanceof CarbonInterface) {
            $immutable = CarbonImmutable::instance($createdAt);
        } else {
            $immutable = CarbonImmutable::parse((string) ($createdAt ?? 'now'));
        }

        $workspaceId = $entry->workspace_id ?? null;
        $toolName = $entry->tool_name ?? null;

        return new self(
            id: isset($entry->id) ? (int) $entry->id : null,
            workspaceId: $workspaceId === null ? null : (string) $workspaceId,
            toolName: $toolName === null ? null : (string) $toolName,
            query: (string) ($entry->query_text ?? ''),
            queryHash: (string) ($entry->query_hash ?? ''),
            isSafe: (bool) ($entry->is_safe ?? false),
            resultCount: isset($entry->result_count) ? (int) $entry->result_count : null,
            durationMs: isset($entry->duration_ms) ? (int) $entry->duration_ms : null,
            metadata: (array) ($entry->metadata ?? []),
            createdAt: $immutable,
        );
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'workspace_id' => $this->workspaceId,
            'tool_name' => $this->toolName,
            'query' => $this->query,
            'query_hash' => $this->queryHash,
            'is_safe' => $this->isSafe,
            'result_count' => $this->resultCount,
            'duration_ms' => $this->durationMs,
            'metadata' => $this->metadata,
            'created_at' => $this->createdAt->toIso8601String(),
        ];
    }
}
