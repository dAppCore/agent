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
 * Update a task's status or notes within a phase.
 *
 * Tasks are stored as a JSON array on the phase model.
 * Handles legacy string-format tasks by normalising to {name, status}.
 *
 * Usage:
 *   $task = UpdateTask::run('deploy-v2', '1', 0, 1, 'in_progress', 'Started build');
 */
class UpdateTask
{
    use Action;

    /**
     * @return array{task: array, plan_progress: array}
     *
     * @throws \InvalidArgumentException
     */
    public function handle(string $planSlug, string|int $phase, int $taskIndex, int $workspaceId, ?string $status = null, ?string $notes = null): array
    {
        if ($status !== null) {
            $valid = ['pending', 'in_progress', 'completed', 'blocked', 'skipped'];
            if (! in_array($status, $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('status must be one of: %s', implode(', ', $valid))
                );
            }
        }

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

        // Normalise legacy string-format tasks
        if (is_string($tasks[$taskIndex])) {
            $tasks[$taskIndex] = ['name' => $tasks[$taskIndex], 'status' => 'pending'];
        }

        if ($status !== null) {
            $tasks[$taskIndex]['status'] = $status;
        }

        if ($notes !== null) {
            $tasks[$taskIndex]['notes'] = $notes;
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
            ->where(function ($query) use ($identifier) {
                $query->where('name', $identifier)
                    ->orWhere('order', $identifier);
            })
            ->first();
    }
}
