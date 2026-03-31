<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sync;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\SyncRecord;
use Illuminate\Support\Carbon;

class PullFleetContext
{
    use Action;

    /**
     * @return array<int, array<string, mixed>>
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, ?string $since = null): array
    {
        if ($agentId === '') {
            throw new \InvalidArgumentException('agent_id is required');
        }

        $query = BrainMemory::query()
            ->forWorkspace($workspaceId)
            ->active()
            ->latest();

        if ($since !== null && $since !== '') {
            $query->where('created_at', '>=', Carbon::parse($since));
        }

        $items = $query->limit(25)->get();

        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if ($node) {
            SyncRecord::create([
                'fleet_node_id' => $node->id,
                'direction' => 'pull',
                'payload_size' => strlen((string) json_encode($items->toArray())),
                'items_count' => $items->count(),
                'synced_at' => now(),
            ]);
        }

        return $items->map(fn (BrainMemory $memory) => $memory->toMcpContext())->values()->all();
    }
}
