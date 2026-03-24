<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\State;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\WorkspaceState;

/**
 * Set a workspace state value.
 */
class StateSet extends AgentTool
{
    protected string $category = 'state';

    protected array $scopes = ['write'];

    /**
     * Get the dependencies for this tool.
     *
     * Workspace context is required to ensure tenant isolation.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required for state operations'),
        ];
    }

    public function name(): string
    {
        return 'state_set';
    }

    public function description(): string
    {
        return 'Set a workspace state value';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'plan_slug' => [
                    'type' => 'string',
                    'description' => 'Plan slug identifier',
                ],
                'key' => [
                    'type' => 'string',
                    'description' => 'State key',
                ],
                'value' => [
                    'type' => ['string', 'number', 'boolean', 'object', 'array'],
                    'description' => 'State value',
                ],
                'category' => [
                    'type' => 'string',
                    'description' => 'State category for organisation',
                ],
            ],
            'required' => ['plan_slug', 'key', 'value'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $planSlug = $this->require($args, 'plan_slug');
            $key = $this->require($args, 'key');
            $value = $this->require($args, 'value');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        // Validate workspace context for tenant isolation
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key and started a session. See: https://host.uk.com/ai');
        }

        // Query plan with workspace scope to prevent cross-tenant access
        $plan = AgentPlan::forWorkspace($workspaceId)
            ->where('slug', $planSlug)
            ->first();

        if (! $plan) {
            return $this->error("Plan not found: {$planSlug}");
        }

        $state = WorkspaceState::updateOrCreate(
            [
                'agent_plan_id' => $plan->id,
                'key' => $key,
            ],
            [
                'value' => $value,
                'category' => $this->optional($args, 'category', 'general'),
            ]
        );

        return $this->success([
            'state' => [
                'key' => $state->key,
                'value' => $state->value,
                'category' => $state->category,
            ],
        ]);
    }
}
