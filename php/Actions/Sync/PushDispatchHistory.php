<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sync;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\SyncRecord;

class PushDispatchHistory
{
    use Action;

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

            $synced++;
        }

        SyncRecord::create([
            'fleet_node_id' => $node->id,
            'direction' => 'push',
            'payload_size' => strlen((string) json_encode($dispatches)),
            'items_count' => count($dispatches),
            'synced_at' => now(),
        ]);

        return ['synced' => $synced];
    }
}
