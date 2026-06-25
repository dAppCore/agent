<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\DispatchJob;
use Illuminate\Support\Collection;

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
                'hostname' => $attributes['hostname'],
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

    /**
     * @param  array<string, mixed>  $attributes
     */
    public function enqueue(int $workspaceId, array $attributes): DispatchJob
    {
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
            'status' => $attributes['status'] ?? DispatchJob::STATUS_PENDING,
            'metadata' => $attributes['metadata'] ?? null,
        ]);
        $job->save();

        return $job->fresh() ?? $job;
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

        $availableSlots = max(0, $registration->max_concurrent - $runningCount);

        if ($availableSlots === 0) {
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
            ->limit(max(50, $availableSlots * 5))
            ->get()
            ->filter(fn (DispatchJob $job): bool => $this->matchesAgent($job, $registration))
            ->values();

        $assigned = collect();

        foreach ($candidates as $job) {
            if ($assigned->count() >= $availableSlots) {
                break;
            }

            // Atomic claim — the conditional update only affects a row for the
            // agent that gets there first. Concurrent installs polling the same
            // workspace queue therefore can't double-claim a job.
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
