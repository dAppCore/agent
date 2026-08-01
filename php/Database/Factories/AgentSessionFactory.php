<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Database\Factories;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Factories\Factory;
use Illuminate\Support\Str;

/**
 * @extends Factory<AgentSession>
 */
class AgentSessionFactory extends Factory
{
    protected $model = AgentSession::class;

    public function definition(): array
    {
        return [
            'workspace_id' => Workspace::factory(),
            'agent_api_key_id' => null,
            'agent_plan_id' => null,
            // agent_sessions.session_id is unique — a random suffix keeps a
            // ->count(n) run from colliding with itself.
            'session_id' => 'sess_'.Str::lower(Str::random(24)),
            'agent_type' => AgentSession::AGENT_SONNET,
            'status' => AgentSession::STATUS_ACTIVE,
            'context_summary' => null,
            'work_log' => null,
            'artifacts' => null,
            'handoff_notes' => null,
            'final_summary' => null,
            'started_at' => now(),
            'last_active_at' => now(),
            'ended_at' => null,
        ];
    }

    public function active(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentSession::STATUS_ACTIVE,
            'ended_at' => null,
        ]);
    }

    public function paused(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentSession::STATUS_PAUSED,
            'ended_at' => null,
        ]);
    }

    public function completed(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentSession::STATUS_COMPLETED,
            'ended_at' => now(),
        ]);
    }

    public function failed(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentSession::STATUS_FAILED,
            'ended_at' => now(),
        ]);
    }

    public function handedOff(): static
    {
        return $this->state(fn (): array => [
            'status' => AgentSession::STATUS_HANDED_OFF,
            'ended_at' => now(),
        ]);
    }

    public function opus(): static
    {
        return $this->state(fn (): array => ['agent_type' => AgentSession::AGENT_OPUS]);
    }

    public function sonnet(): static
    {
        return $this->state(fn (): array => ['agent_type' => AgentSession::AGENT_SONNET]);
    }

    public function haiku(): static
    {
        return $this->state(fn (): array => ['agent_type' => AgentSession::AGENT_HAIKU]);
    }

    /**
     * Bind the session to an existing plan — and to that plan's workspace, so
     * the session is not silently scoped to a second, freshly-made one.
     */
    public function forPlan(AgentPlan $plan): static
    {
        return $this->state(fn (): array => [
            'agent_plan_id' => $plan->id,
            'workspace_id' => $plan->workspace_id,
        ]);
    }
}
