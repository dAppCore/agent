<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Database\Eloquent\Collection;

class ListNodes
{
    use Action;

    public function handle(int $workspaceId, ?string $status = null, ?string $platform = null): Collection
    {
        $query = FleetNode::query()->where('workspace_id', $workspaceId);

        if ($status !== null && $status !== '') {
            $query->where('status', $status);
        }

        if ($platform !== null && $platform !== '') {
            $query->where('platform', $platform);
        }

        return $query->orderBy('agent_id')->get();
    }
}
