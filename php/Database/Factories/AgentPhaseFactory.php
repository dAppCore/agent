<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Database\Factories;

use Core\Mod\Agentic\Models\AgentPhase;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<AgentPhase>
 */
class AgentPhaseFactory extends Factory
{
    protected $model = AgentPhase::class;

    public function definition(): array
    {
        return [
            'agent_plan_id' => AgentPlanFactory::new(),
            'order' => $this->faker->numberBetween(1, 10),
            'name' => ucfirst($this->faker->words(2, true)),
            'description' => $this->faker->sentence(),
            'tasks' => null,
            'dependencies' => null,
            'status' => AgentPhase::STATUS_PENDING,
            'completion_criteria' => null,
            'started_at' => null,
            'completed_at' => null,
            'metadata' => null,
        ];
    }

    public function pending(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentPhase::STATUS_PENDING,
            'started_at' => null,
            'completed_at' => null,
        ]);
    }

    /**
     * started_at is set here, not just the status: the phase model and the
     * services that advance it read the timestamps to decide what is running,
     * so a state that only moved the string would be a half-built phase.
     */
    public function inProgress(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentPhase::STATUS_IN_PROGRESS,
            'started_at' => now(),
            'completed_at' => null,
        ]);
    }

    public function completed(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentPhase::STATUS_COMPLETED,
            'started_at' => now()->subHour(),
            'completed_at' => now(),
        ]);
    }

    public function blocked(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentPhase::STATUS_BLOCKED,
            'started_at' => now()->subHour(),
            'completed_at' => null,
        ]);
    }

    public function skipped(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentPhase::STATUS_SKIPPED,
            'started_at' => null,
            'completed_at' => null,
        ]);
    }
}
