<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Data;

use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;

final readonly class CreditTransaction
{
    public function __construct(
        public ?int $id,
        public int $workspaceId,
        public ?int $fleetNodeId,
        public string $taskType,
        public int $amount,
        public int $balanceAfter,
        public ?string $description,
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

        return new self(
            id: isset($entry->id) ? (int) $entry->id : null,
            workspaceId: (int) ($entry->workspace_id ?? 0),
            fleetNodeId: isset($entry->fleet_node_id) ? (int) $entry->fleet_node_id : null,
            taskType: (string) ($entry->task_type ?? ''),
            amount: (int) ($entry->amount ?? 0),
            balanceAfter: (int) ($entry->balance_after ?? 0),
            description: isset($entry->description) ? (string) $entry->description : null,
            createdAt: $immutable,
        );
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'workspace_id' => $this->workspaceId,
            'fleet_node_id' => $this->fleetNodeId,
            'task_type' => $this->taskType,
            'amount' => $this->amount,
            'balance_after' => $this->balanceAfter,
            'description' => $this->description,
            'created_at' => $this->createdAt->toIso8601String(),
        ];
    }
}
