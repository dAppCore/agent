<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Database\Factories;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Factories\Factory;
use Illuminate\Support\Str;

/**
 * @extends Factory<AgentPlan>
 */
class AgentPlanFactory extends Factory
{
    protected $model = AgentPlan::class;

    public function definition(): array
    {
        $title = ucfirst($this->faker->words(3, true));

        return [
            'workspace_id' => Workspace::factory(),
            'slug' => Str::slug($title).'-'.Str::lower(Str::random(6)),
            'title' => $title,
            'description' => $this->faker->sentence(),
            'context' => null,
            'phases' => null,
            'status' => AgentPlan::STATUS_DRAFT,
            'current_phase' => null,
            'metadata' => null,
            'source_file' => null,
        ];
    }

    public function draft(): static
    {
        return $this->state(fn (): array => ['status' => AgentPlan::STATUS_DRAFT]);
    }

    public function active(): static
    {
        return $this->state(fn (): array => ['status' => AgentPlan::STATUS_ACTIVE]);
    }

    public function completed(): static
    {
        return $this->state(fn (): array => ['status' => AgentPlan::STATUS_COMPLETED]);
    }

    public function archived(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now(),
        ]);
    }

    /**
     * Attach $count phases, ordered 1..$count, once the plan exists.
     *
     * afterCreating rather than a state: agent_phases.agent_plan_id is a
     * constrained foreign key, so the phases cannot be built until the plan
     * has an id.
     */
    public function withPhases(int $count = 3): static
    {
        return $this->afterCreating(function (AgentPlan $plan) use ($count): void {
            AgentPhaseFactory::new()
                ->count($count)
                ->sequence(fn ($sequence): array => ['order' => $sequence->index + 1])
                ->create(['agent_plan_id' => $plan->id]);
        });
    }
}
