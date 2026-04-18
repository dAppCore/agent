<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;

class RegisterNode
{
    use Action;

    /**
     * @param  array<string>  $models
     * @param  array<string, mixed>  $capabilities
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, string $platform, array $models = [], array $capabilities = []): FleetNode
    {
        if ($workspaceId <= 0) {
            throw new \InvalidArgumentException('workspace_id is required');
        }

        if ($agentId === '') {
            throw new \InvalidArgumentException('agent_id is required');
        }

        return FleetNode::updateOrCreate(
            ['agent_id' => $agentId],
            [
                'workspace_id' => $workspaceId,
                'platform' => $platform !== '' ? $platform : 'unknown',
                'models' => $models,
                'capabilities' => $capabilities,
                'status' => FleetNode::STATUS_ONLINE,
                'registered_at' => now(),
                'last_heartbeat_at' => now(),
            ],
        );
    }
}
