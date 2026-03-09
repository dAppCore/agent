<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Plan;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Illuminate\Support\Str;

/**
 * Create a new work plan with phases and tasks.
 *
 * Validates input, generates a unique slug, creates the plan
 * and any associated phases with their tasks.
 *
 * Usage:
 *   $plan = CreatePlan::run([
 *       'title' => 'Deploy v2',
 *       'phases' => [['name' => 'Build', 'tasks' => ['compile', 'test']]],
 *   ], 1);
 */
class CreatePlan
{
    use Action;

    /**
     * @param  array{title: string, slug?: string, description?: string, context?: array, phases?: array}  $data
     *
     * @throws \InvalidArgumentException
     */
    public function handle(array $data, int $workspaceId): AgentPlan
    {
        $title = $data['title'] ?? null;
        if (! is_string($title) || $title === '' || mb_strlen($title) > 255) {
            throw new \InvalidArgumentException('title is required and must be a non-empty string (max 255 characters)');
        }

        $slug = $data['slug'] ?? null;
        if ($slug !== null) {
            if (! is_string($slug) || mb_strlen($slug) > 255) {
                throw new \InvalidArgumentException('slug must be a string (max 255 characters)');
            }
        } else {
            $slug = Str::slug($title).'-'.Str::random(6);
        }

        if (AgentPlan::where('slug', $slug)->exists()) {
            throw new \InvalidArgumentException("Plan with slug '{$slug}' already exists");
        }

        $plan = AgentPlan::create([
            'slug' => $slug,
            'title' => $title,
            'description' => $data['description'] ?? null,
            'status' => AgentPlan::STATUS_DRAFT,
            'context' => $data['context'] ?? [],
            'workspace_id' => $workspaceId,
        ]);

        if (! empty($data['phases'])) {
            foreach ($data['phases'] as $order => $phaseData) {
                $tasks = collect($phaseData['tasks'] ?? [])->map(fn ($task) => [
                    'name' => $task,
                    'status' => 'pending',
                ])->all();

                AgentPhase::create([
                    'agent_plan_id' => $plan->id,
                    'name' => $phaseData['name'] ?? 'Phase '.($order + 1),
                    'description' => $phaseData['description'] ?? null,
                    'order' => $order + 1,
                    'status' => AgentPhase::STATUS_PENDING,
                    'tasks' => $tasks,
                ]);
            }
        }

        return $plan->load('agentPhases');
    }
}
