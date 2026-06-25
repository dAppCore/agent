<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\DispatchJob;
use Illuminate\Support\Collection;

/**
 * The single fleet service: agent registration + a workspace-scoped, claim-based
 * dispatch queue. Used by System A (console + MCP register/checkin/dispatch) and
 * System B (the Go core-agent fleet/sync endpoints) so one workspace shares one
 * queue across a group of installs.
 */
class DispatchService
{
    /**
     * @param  array<string, mixed>  $attributes
     */
    public function register(int $workspaceId, array $attributes): AgentRegistration
    {
        /** @var AgentRegistration $registration */
        $registration = AgentRegistration::query()->updateOrCreate(
            [
                'workspace_id' => $workspaceId,
                'agent_id' => $attributes['agent_id'],
            ],
            [
                'hostname' => $attributes['hostname'] ?? $attributes['agent_id'],
                'platform' => $attributes['platform'] ?? null,
                'capabilities' => $attributes['capabilities'] ?? [],
                'models' => $attributes['models'] ?? null,
                'compute_budget' => $attributes['compute_budget'] ?? null,
                'max_concurrent' => (int) ($attributes['max_concurrent'] ?? 1),
                'labels' => $attributes['labels'] ?? [],
                'version' => $attributes['version'] ?? null,
                'status' => $attributes['status'] ?? AgentRegistration::STATUS_ONLINE,
                'metadata' => $attributes['metadata'] ?? null,
                'connected_at' => now(),
                'last_heartbeat_at' => now(),
            ]
        );

        return $registration->fresh() ?? $registration;
    }

    public function findRegistration(int $workspaceId, string $agentId): ?AgentRegistration
    {
        return AgentRegistration::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();
    }

    public function heartbeat(int $workspaceId, string $agentId, string $status, array $computeBudget = []): ?AgentRegistration
    {
        $registration = $this->findRegistration($workspaceId, $agentId);

        if (! $registration instanceof AgentRegistration) {
            return null;
        }

        $registration->forceFill([
            'status' => $status,
            'compute_budget' => $computeBudget !== [] ? $computeBudget : $registration->compute_budget,
            'last_heartbeat_at' => now(),
        ])->save();

        return $registration->fresh() ?? $registration;
    }

    public function deregister(int $workspaceId, string $agentId): void
    {
        AgentRegistration::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->update([
                'status' => AgentRegistration::STATUS_OFFLINE,
                'last_heartbeat_at' => now(),
            ]);
    }

    /**
     * @return Collection<int, AgentRegistration>
     */
    public function listAgents(int $workspaceId, ?string $status = null, ?string $platform = null): Collection
    {
        $query = AgentRegistration::query()->where('workspace_id', $workspaceId);

        if ($status !== null && $status !== '') {
            $query->where('status', $status);
        }

        if ($platform !== null && $platform !== '') {
            $query->where('platform', $platform);
        }

        return $query->orderByDesc('last_heartbeat_at')->get();
    }

    /**
     * Fleet stats — preserves the /v1/fleet/stats contract (nodes_online,
     * tasks_today, tasks_week, repos_touched, findings_total) computed from the
     * unified tables, plus the queue counters.
     *
     * @return array<string, int>
     */
    public function stats(int $workspaceId): array
    {
        $jobs = DispatchJob::query()->where('workspace_id', $workspaceId);

        $findingsTotal = (clone $jobs)->whereNotNull('findings')->get(['findings'])
            ->sum(fn (DispatchJob $job): int => is_array($job->findings) ? count($job->findings) : 0);

        return [
            'nodes_online' => AgentRegistration::query()->where('workspace_id', $workspaceId)->where('status', AgentRegistration::STATUS_ONLINE)->count(),
            'nodes_total' => AgentRegistration::query()->where('workspace_id', $workspaceId)->count(),
            'tasks_today' => (clone $jobs)->whereDate('created_at', today())->count(),
            'tasks_week' => (clone $jobs)->where('created_at', '>=', now()->subDays(7))->count(),
            'repos_touched' => (clone $jobs)->distinct()->count('repo'),
            'findings_total' => (int) $findingsTotal,
            'pending' => (clone $jobs)->pending()->count(),
            'running' => (clone $jobs)->active()->count(),
            'completed' => (clone $jobs)->where('status', DispatchJob::STATUS_COMPLETED)->count(),
            'failed' => (clone $jobs)->where('status', DispatchJob::STATUS_FAILED)->count(),
        ];
    }

    /**
     * @param  array<string, mixed>  $attributes
     */
    public function enqueue(int $workspaceId, array $attributes): DispatchJob
    {
        $assignedAgent = $attributes['assigned_agent'] ?? null;

        $job = new DispatchJob;
        $job->forceFill([
            'workspace_id' => $workspaceId,
            'created_by' => $attributes['created_by'] ?? null,
            'repo' => $attributes['repo'],
            'org' => $attributes['org'] ?? null,
            'task' => $attributes['task'],
            'agent_type' => $attributes['agent_type'] ?? null,
            'template' => $attributes['template'] ?? null,
            'branch' => $attributes['branch'] ?? null,
            'priority' => (int) ($attributes['priority'] ?? 5),
            'labels' => $attributes['labels'] ?? [],
            'status' => $attributes['status'] ?? ($assignedAgent ? DispatchJob::STATUS_ASSIGNED : DispatchJob::STATUS_PENDING),
            'assigned_agent' => $assignedAgent,
            'assigned_at' => $assignedAgent ? now() : null,
            'findings' => $attributes['findings'] ?? null,
            'changes' => $attributes['changes'] ?? null,
            'report' => $attributes['report'] ?? null,
            'metadata' => $attributes['metadata'] ?? null,
        ]);
        $job->save();

        return $job->fresh() ?? $job;
    }

    /**
     * Mark a job done (or failed) and write the agent's report back — the
     * completion path System A never had a clean endpoint for.
     *
     * @param  array<string, mixed>  $data
     */
    public function complete(int $workspaceId, string $agentId, string $jobId, array $data = []): ?DispatchJob
    {
        $job = DispatchJob::query()
            ->where('workspace_id', $workspaceId)
            ->where('assigned_agent', $agentId)
            ->whereKey($jobId)
            ->first();

        if (! $job instanceof DispatchJob) {
            return null;
        }

        $job->forceFill([
            'status' => $data['status'] ?? DispatchJob::STATUS_COMPLETED,
            'result' => $data['result'] ?? $job->result,
            'findings' => $data['findings'] ?? $job->findings,
            'changes' => $data['changes'] ?? $job->changes,
            'report' => $data['report'] ?? $job->report,
            'completed_at' => now(),
        ])->save();

        AgentRegistration::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->where('current_task_id', $jobId)
            ->update(['current_task_id' => null]);

        return $job->fresh();
    }

    /**
     * @return array{registration: AgentRegistration|null, jobs: Collection<int, DispatchJob>}
     */
    public function checkIn(int $workspaceId, string $agentId): array
    {
        $registration = $this->findRegistration($workspaceId, $agentId);

        if (! $registration instanceof AgentRegistration) {
            return [
                'registration' => null,
                'jobs' => collect(),
            ];
        }

        $registration->forceFill([
            'status' => AgentRegistration::STATUS_ONLINE,
            'last_heartbeat_at' => now(),
        ])->save();

        return [
            'registration' => $registration->fresh() ?? $registration,
            'jobs' => $this->assignJobs($registration),
        ];
    }

    /**
     * Claim and return a single next job for an agent (the fleet `nextTask`
     * loop), respecting max_concurrent. Heartbeats the agent as a side effect.
     *
     * @param  array<int, string>  $capabilities
     */
    public function nextTask(int $workspaceId, string $agentId, array $capabilities = []): ?DispatchJob
    {
        $registration = $this->findRegistration($workspaceId, $agentId);

        if (! $registration instanceof AgentRegistration) {
            return null;
        }

        $registration->forceFill([
            'status' => AgentRegistration::STATUS_ONLINE,
            'last_heartbeat_at' => now(),
        ])->save();

        $running = DispatchJob::query()
            ->where('workspace_id', $workspaceId)
            ->where('assigned_agent', $agentId)
            ->active()
            ->count();

        if ($registration->max_concurrent - $running <= 0) {
            return null;
        }

        $job = $this->claimUpTo($registration, 1)->first();

        if ($job instanceof DispatchJob) {
            $registration->forceFill(['current_task_id' => $job->id])->save();
        }

        return $job;
    }

    /**
     * @return Collection<int, DispatchJob>
     */
    public function listJobs(int $workspaceId, ?string $status = null): Collection
    {
        $query = DispatchJob::query()
            ->where('workspace_id', $workspaceId)
            ->orderByDesc('priority')
            ->orderBy('created_at');

        if ($status !== null && $status !== '') {
            $query->where('status', $status);
        }

        /** @var Collection<int, DispatchJob> $jobs */
        $jobs = $query->get();

        return $jobs;
    }

    /**
     * @return Collection<int, DispatchJob>
     */
    private function assignJobs(AgentRegistration $registration): Collection
    {
        if ($registration->status !== AgentRegistration::STATUS_ONLINE) {
            return collect();
        }

        $runningCount = DispatchJob::query()
            ->where('workspace_id', $registration->workspace_id)
            ->where('assigned_agent', $registration->agent_id)
            ->active()
            ->count();

        return $this->claimUpTo($registration, max(0, $registration->max_concurrent - $runningCount));
    }

    /**
     * Atomically claim up to $limit pending jobs for the agent. The conditional
     * update only affects a row for the agent that gets there first, so
     * concurrent installs polling the same workspace queue can't double-claim.
     *
     * @return Collection<int, DispatchJob>
     */
    private function claimUpTo(AgentRegistration $registration, int $limit): Collection
    {
        if ($limit <= 0) {
            return collect();
        }

        // Fetch headroom beyond the open slots so lost races (a peer claimed
        // first) still leave enough candidates to fill this agent's capacity.
        /** @var Collection<int, DispatchJob> $candidates */
        $candidates = DispatchJob::query()
            ->where('workspace_id', $registration->workspace_id)
            ->pending()
            ->orderByDesc('priority')
            ->orderBy('created_at')
            ->limit(max(50, $limit * 5))
            ->get()
            ->filter(fn (DispatchJob $job): bool => $this->matchesAgent($job, $registration))
            ->values();

        $assigned = collect();

        foreach ($candidates as $job) {
            if ($assigned->count() >= $limit) {
                break;
            }

            $claimed = DispatchJob::query()
                ->whereKey($job->getKey())
                ->where('status', DispatchJob::STATUS_PENDING)
                ->whereNull('assigned_agent')
                ->update([
                    'status' => DispatchJob::STATUS_ASSIGNED,
                    'assigned_agent' => $registration->agent_id,
                    'assigned_at' => now(),
                ]);

            if ($claimed === 1) {
                $fresh = $job->fresh();

                if ($fresh !== null) {
                    $assigned->push($fresh);
                }
            }
        }

        /** @var Collection<int, DispatchJob> $assigned */
        return $assigned;
    }

    private function matchesAgent(DispatchJob $job, AgentRegistration $registration): bool
    {
        return $registration->hasCapability($job->agent_type)
            && $registration->hasLabels($job->labels ?? []);
    }
}
