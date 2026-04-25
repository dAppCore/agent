<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sync;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\SyncRecord;
use Core\Mod\Agentic\Models\WorkspaceState;

class PushDispatchHistory
{
    use Action;

    private const SYNC_CATEGORY = 'sync';

    /**
     * @param  array<int, array<string, mixed>>  $dispatches
     * @return array{synced: int}
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, array $dispatches): array
    {
        if ($agentId === '') {
            throw new \InvalidArgumentException('agent_id is required');
        }

        $node = FleetNode::firstOrCreate(
            ['agent_id' => $agentId],
            [
                'workspace_id' => $workspaceId,
                'platform' => 'remote',
                'status' => FleetNode::STATUS_ONLINE,
                'registered_at' => now(),
                'last_heartbeat_at' => now(),
            ],
        );

        $synced = 0;
        $planUpdates = [];

        foreach ($dispatches as $dispatch) {
            $repo = (string) ($dispatch['repo'] ?? '');
            $status = (string) ($dispatch['status'] ?? 'completed');
            $workspace = (string) ($dispatch['workspace'] ?? '');
            $task = (string) ($dispatch['task'] ?? '');

            if ($repo === '' && $workspace === '') {
                continue;
            }

            BrainMemory::create([
                'workspace_id' => $workspaceId,
                'agent_id' => $agentId,
                'type' => 'observation',
                'content' => trim("Repo: {$repo}\nWorkspace: {$workspace}\nStatus: {$status}\nTask: {$task}"),
                'tags' => array_values(array_filter([
                    'sync',
                    $repo !== '' ? $repo : null,
                    $status,
                ])),
                'project' => $repo !== '' ? $repo : null,
                'confidence' => 0.7,
                'source' => 'sync.push',
            ]);

            $planUpdate = $this->resolvePlanUpdate($dispatch, $status);
            if ($planUpdate !== null) {
                $planUpdates[$planUpdate['plan_id']] = $planUpdate;
            }

            $synced++;
        }

        SyncRecord::create([
            'fleet_node_id' => $node->id,
            'direction' => 'push',
            'payload_size' => strlen((string) json_encode($dispatches)),
            'items_count' => count($dispatches),
            'synced_at' => now(),
        ]);

        $dispatchAt = now()->toIso8601String();

        foreach ($planUpdates as $planUpdate) {
            $this->writeSyncState(
                $planUpdate['plan_id'],
                'sync.last_dispatch_at',
                $dispatchAt,
                'Most recent dispatch sync timestamp.',
            );
            $this->writeSyncState(
                $planUpdate['plan_id'],
                'sync.last_agent_type',
                $planUpdate['agent_type'],
                'Most recent synced agent type.',
            );
            $this->writeSyncState(
                $planUpdate['plan_id'],
                'sync.last_findings_count',
                $planUpdate['findings_count'],
                'Most recent synced findings count.',
            );
            $this->writeSyncState(
                $planUpdate['plan_id'],
                'sync.last_status',
                $planUpdate['status'],
                'Most recent synced dispatch status.',
            );
        }

        // TODO: subscriber notification — no notifier interface yet, out of scope for this ticket

        return ['synced' => $synced];
    }

    /**
     * @param  array<string, mixed>  $dispatch
     * @return array{plan_id: int, agent_type: string, findings_count: int, status: string}|null
     */
    private function resolvePlanUpdate(array $dispatch, string $status): ?array
    {
        $plan = $this->resolvePlan($dispatch);
        if (! $plan instanceof AgentPlan) {
            return null;
        }

        $findings = $dispatch['findings'] ?? [];

        return [
            'plan_id' => $plan->id,
            'agent_type' => (string) ($dispatch['agent_type'] ?? ''),
            'findings_count' => is_array($findings) ? count($findings) : 0,
            'status' => $status,
        ];
    }

    /**
     * @param  array<string, mixed>  $dispatch
     */
    private function resolvePlan(array $dispatch): ?AgentPlan
    {
        $planId = (int) ($dispatch['agent_plan_id'] ?? 0);
        if ($planId > 0) {
            $plan = AgentPlan::find($planId);
            if ($plan instanceof AgentPlan) {
                return $plan;
            }
        }

        $planSlug = trim((string) ($dispatch['plan_slug'] ?? ''));
        if ($planSlug === '') {
            return null;
        }

        return AgentPlan::where('slug', $planSlug)->first();
    }

    private function writeSyncState(int $planId, string $key, mixed $value, string $description): void
    {
        $state = WorkspaceState::set($planId, $key, $value, WorkspaceState::TYPE_JSON);
        $state->forceFill([
            'category' => self::SYNC_CATEGORY,
            'description' => $description,
        ])->save();
    }
}
