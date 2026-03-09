<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\State;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentPlan;

/**
 * List all state values for a plan.
 */
class StateList extends AgentTool
{
    protected string $category = 'state';

    protected array $scopes = ['read'];

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
        return 'state_list';
    }

    public function description(): string
    {
        return 'List all state values for a plan';
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
                'category' => [
                    'type' => 'string',
                    'description' => 'Filter by category',
                ],
            ],
            'required' => ['plan_slug'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $planSlug = $this->require($args, 'plan_slug');
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

        $query = $plan->states();

        $category = $this->optional($args, 'category');
        if (! empty($category)) {
            $query->where('category', $category);
        }

        $states = $query->get();

        return $this->success([
            'states' => $states->map(fn ($state) => [
                'key' => $state->key,
                'value' => $state->value,
                'category' => $state->category,
            ])->all(),
            'total' => $states->count(),
        ]);
    }
}
