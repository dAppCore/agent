<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Data;

final readonly class FleetStats
{
    public function __construct(
        public int $nodesOnline,
        public int $tasksToday,
        public int $tasksWeek,
        public int $reposTouched,
        public int $findingsTotal,
        public int $computeHours,
    ) {}

    public static function fromArray(array $stats): self
    {
        return new self(
            nodesOnline: (int) ($stats['nodes_online'] ?? 0),
            tasksToday: (int) ($stats['tasks_today'] ?? 0),
            tasksWeek: (int) ($stats['tasks_week'] ?? 0),
            reposTouched: (int) ($stats['repos_touched'] ?? 0),
            findingsTotal: (int) ($stats['findings_total'] ?? 0),
            computeHours: (int) ($stats['compute_hours'] ?? 0),
        );
    }

    public function toArray(): array
    {
        return [
            'nodes_online' => $this->nodesOnline,
            'tasks_today' => $this->tasksToday,
            'tasks_week' => $this->tasksWeek,
            'repos_touched' => $this->reposTouched,
            'findings_total' => $this->findingsTotal,
            'compute_hours' => $this->computeHours,
        ];
    }
}
