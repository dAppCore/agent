<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Task;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;

/**
 * Toggle a task's completion status (pending <-> completed).
 *
 * Quick convenience method for marking tasks done or undone.
 *
 * Usage:
 *   $result = ToggleTask::run('deploy-v2', '1', 0, 1);
 */
class ToggleTask
{
    use Action;

    /**
     * @return array{task: array, plan_progress: array}
     *
     * @throws \InvalidArgumentException
     */
    public function handle(string $planSlug, string|int $phase, int $taskIndex, int $workspaceId): array
    {
        $plan = AgentPlan::forWorkspace($workspaceId)
            ->where('slug', $planSlug)
            ->first();

        if (! $plan) {
            throw new \InvalidArgumentException("Plan not found: {$planSlug}");
        }

        $resolved = $this->resolvePhase($plan, $phase);

        if (! $resolved) {
            throw new \InvalidArgumentException("Phase not found: {$phase}");
        }

        $tasks = $resolved->tasks ?? [];

        if (! isset($tasks[$taskIndex])) {
            throw new \InvalidArgumentException("Task not found at index: {$taskIndex}");
        }

        $currentStatus = is_string($tasks[$taskIndex])
            ? 'pending'
            : ($tasks[$taskIndex]['status'] ?? 'pending');

        $newStatus = $currentStatus === 'completed' ? 'pending' : 'completed';

        if (is_string($tasks[$taskIndex])) {
            $tasks[$taskIndex] = [
                'name' => $tasks[$taskIndex],
                'status' => $newStatus,
            ];
        } else {
            $tasks[$taskIndex]['status'] = $newStatus;
        }

        $resolved->update(['tasks' => $tasks]);

        return [
            'task' => $tasks[$taskIndex],
            'plan_progress' => $plan->fresh()->getProgress(),
        ];
    }

    private function resolvePhase(AgentPlan $plan, string|int $identifier): ?AgentPhase
    {
        if (is_numeric($identifier)) {
            return $plan->agentPhases()->where('order', (int) $identifier)->first();
        }

        return $plan->agentPhases()
            ->where('name', $identifier)
            ->first();
    }
}
