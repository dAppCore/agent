<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Auth;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;

class RevokeAgentKey
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, int $keyId): bool
    {
        $key = AgentApiKey::query()
            ->where('workspace_id', $workspaceId)
            ->find($keyId);

        if (! $key) {
            throw new \InvalidArgumentException('API key not found');
        }

        app(AgentApiKeyService::class)->revoke($key);

        return true;
    }
}
