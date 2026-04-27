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
        if ($createdAt === null) {
            throw new \InvalidArgumentException('CreditTransaction requires a created_at value.');
        }

        if ($createdAt instanceof CarbonImmutable) {
            $immutable = $createdAt;
        } elseif ($createdAt instanceof CarbonInterface) {
            $immutable = CarbonImmutable::instance($createdAt);
        } else {
            try {
                $immutable = CarbonImmutable::parse((string) $createdAt);
            } catch (\Throwable) {
                throw new \InvalidArgumentException('CreditTransaction requires a valid created_at value.');
            }
        }

        return new self(
            id: self::optionalInt($entry->id ?? null, 'id'),
            workspaceId: self::requireInt($entry->workspace_id ?? null, 'workspace_id'),
            fleetNodeId: self::optionalInt($entry->fleet_node_id ?? null, 'fleet_node_id'),
            taskType: self::requireString($entry->task_type ?? null, 'task_type'),
            amount: self::requireInt($entry->amount ?? null, 'amount'),
            balanceAfter: self::requireInt($entry->balance_after ?? null, 'balance_after'),
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

    private static function requireInt(mixed $value, string $field): int
    {
        if ($value === null) {
            throw new \InvalidArgumentException(sprintf(
                'CreditTransaction requires %s.',
                $field,
            ));
        }

        return self::coerceInt($value, $field);
    }

    private static function optionalInt(mixed $value, string $field): ?int
    {
        if ($value === null) {
            return null;
        }

        return self::coerceInt($value, $field);
    }

    private static function coerceInt(mixed $value, string $field): int
    {
        if (is_int($value)) {
            return $value;
        }

        if (is_float($value) && floor($value) === $value) {
            return (int) $value;
        }

        if (is_string($value) && preg_match('/^-?\d+$/', $value) === 1) {
            return (int) $value;
        }

        throw new \InvalidArgumentException(sprintf(
            'CreditTransaction requires integer %s.',
            $field,
        ));
    }

    private static function requireString(mixed $value, string $field): string
    {
        if (! is_string($value) || trim($value) === '') {
            throw new \InvalidArgumentException(sprintf(
                'CreditTransaction requires non-empty %s.',
                $field,
            ));
        }

        return $value;
    }
}
