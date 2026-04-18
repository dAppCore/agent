<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Auth;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Illuminate\Support\Carbon;

class ProvisionAgentKey
{
    use Action;

    /**
     * @param  array<string>  $permissions
     *
     * @throws \InvalidArgumentException
     */
    public function handle(
        int $workspaceId,
        string $oauthUserId,
        ?string $name = null,
        array $permissions = [],
        int $rateLimit = 100,
        ?string $expiresAt = null
    ): AgentApiKey {
        if ($workspaceId <= 0) {
            throw new \InvalidArgumentException('workspace_id is required');
        }

        if ($oauthUserId === '') {
            throw new \InvalidArgumentException('oauth_user_id is required');
        }

        $service = app(AgentApiKeyService::class);

        return $service->create(
            $workspaceId,
            $name ?: 'agent-'.$oauthUserId,
            $permissions,
            $rateLimit,
            $expiresAt ? Carbon::parse($expiresAt) : null,
        );
    }
}
