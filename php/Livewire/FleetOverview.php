<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Livewire;

use Core\Mod\Agentic\Actions\Fleet\AssignTask;
use Core\Mod\Agentic\Actions\Fleet\GetFleetStats;
use Core\Mod\Agentic\Actions\Fleet\ListNodes;
use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Validation\Rule;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;

#[Title('Fleet Overview')]
#[Layout('hub::admin.layouts.app')]
/**
 * Inspect workspace fleet state and dispatch new tasks to agents.
 *
 * @example
 * Livewire::test(FleetOverview::class, ['workspaceId' => 7])->call('refreshOverview');
 */
class FleetOverview extends HubComponent
{
    public int $workspaceId = 0;

    public string $statusFilter = '';

    public string $platformFilter = '';

    public string $dispatchAgentId = '';

    public string $dispatchRepo = '';

    public string $dispatchTask = '';

    public string $dispatchBranch = 'dev';

    public string $dispatchTemplate = '';

    public string $dispatchModel = '';

    /**
     * Initialise the fleet screen with access checks and a default agent.
     *
     * @example
     * $component->mount(7);
     */
    public function mount(?int $workspaceId = null): void
    {
        $this->checkHadesAccess();
        $this->workspaceId = $workspaceId ?? $this->resolveWorkspaceId();
        $this->syncDispatchAgentId();
    }

    #[Computed]
    /**
     * Return the fleet nodes visible for the current workspace filters.
     *
     * @example
     * $nodes = $component->nodes();
     */
    public function nodes(): array
    {
        if ($this->workspaceId <= 0) {
            return [];
        }

        try {
            return ListNodes::run($this->workspaceId, $this->statusFilter, $this->platformFilter)
                ->load('currentTask')
                ->map(function (FleetNode $node): array {
                    $currentTask = $node->currentTask;

                    return [
                        'id' => $node->id,
                        'agent_id' => $node->agent_id,
                        'platform' => $node->platform,
                        'models' => $node->models ?? [],
                        'capabilities' => $node->capabilities ?? [],
                        'status' => $node->status,
                        'current_task_id' => $node->current_task_id,
                        'current_task_label' => $currentTask?->repo
                            ?? ($node->current_task_id ? 'Task #'.$node->current_task_id : 'Idle'),
                        'compute_budget' => $node->compute_budget ?? [],
                        'compute_budget_label' => $this->summariseBudget($node->compute_budget ?? []),
                        'last_heartbeat_at' => $node->last_heartbeat_at?->toDateTimeString(),
                        'last_heartbeat_human' => $node->last_heartbeat_at?->diffForHumans() ?? 'Never',
                    ];
                })
                ->values()
                ->all();
        } catch (\Throwable) {
            return [];
        }
    }

    #[Computed]
    /**
     * Return aggregated fleet statistics for the current workspace.
     *
     * @example
     * $stats = $component->stats();
     */
    public function stats(): array
    {
        if ($this->workspaceId <= 0) {
            return [
                'nodes_online' => 0,
                'tasks_today' => 0,
                'tasks_week' => 0,
                'repos_touched' => 0,
                'findings_total' => 0,
                'compute_hours' => 0,
                'nodes_total' => 0,
                'nodes_busy' => 0,
                'nodes_idle' => 0,
            ];
        }

        try {
            $stats = GetFleetStats::run($this->workspaceId);
        } catch (\Throwable) {
            $stats = [
                'nodes_online' => 0,
                'tasks_today' => 0,
                'tasks_week' => 0,
                'repos_touched' => 0,
                'findings_total' => 0,
                'compute_hours' => 0,
            ];
        }

        $nodes = collect($this->nodes);

        return $stats + [
            'nodes_total' => $nodes->count(),
            'nodes_busy' => $nodes->where('status', FleetNode::STATUS_BUSY)->count(),
            'nodes_idle' => $nodes->where('status', FleetNode::STATUS_ONLINE)->count(),
        ];
    }

    #[Computed]
    /**
     * Return the list of platforms present in the current workspace fleet.
     *
     * @example
     * $platforms = $component->platforms();
     */
    public function platforms(): array
    {
        if ($this->workspaceId <= 0) {
            return [];
        }

        try {
            return FleetNode::query()
                ->where('workspace_id', $this->workspaceId)
                ->orderBy('platform')
                ->pluck('platform')
                ->filter(static fn (mixed $platform): bool => is_string($platform) && $platform !== '')
                ->unique()
                ->values()
                ->all();
        } catch (\Throwable) {
            return [];
        }
    }

    /**
     * Pre-fill the dispatch form for a chosen agent.
     *
     * @example
     * Livewire::test(FleetOverview::class, ['workspaceId' => 7])->call('stageDispatch', 'agent-6');
     */
    public function stageDispatch(string $agentId): void
    {
        $this->dispatchAgentId = $agentId;
        $this->toast('Dispatch Ready', "Prepared dispatch for {$agentId}.", 'info');
    }

    /**
     * Validate and queue a new task dispatch for the selected agent.
     *
     * @example
     * Livewire::test(FleetOverview::class, ['workspaceId' => 7])->call('dispatchTask');
     */
    public function dispatchTask(): void
    {
        $this->validate([
            'workspaceId' => 'required|integer|min:1',
            'dispatchAgentId' => [
                'required',
                'string',
                'max:255',
                Rule::exists(FleetNode::class, 'agent_id')->where(
                    fn ($query) => $query->where('workspace_id', $this->workspaceId),
                ),
            ],
            'dispatchRepo' => 'required|string|max:255',
            'dispatchTask' => 'required|string|max:10000',
            'dispatchBranch' => 'nullable|string|max:255',
            'dispatchTemplate' => 'nullable|string|max:255',
            'dispatchModel' => 'nullable|string|max:255',
        ]);

        AssignTask::run(
            $this->workspaceId,
            $this->dispatchAgentId,
            $this->dispatchTask,
            $this->dispatchRepo,
            $this->dispatchTemplate !== '' ? $this->dispatchTemplate : null,
            $this->dispatchBranch !== '' ? $this->dispatchBranch : null,
            $this->dispatchModel !== '' ? $this->dispatchModel : null,
        );

        $agentId = $this->dispatchAgentId;

        $this->dispatchTask = '';
        $this->dispatchTemplate = '';
        $this->dispatchModel = '';
        $this->refreshOverview();

        $this->toast('Task Dispatched', "Queued work for {$agentId}.", 'success');
    }

    /**
     * Refresh computed fleet data and resynchronise the dispatch target.
     *
     * @example
     * Livewire::test(FleetOverview::class, ['workspaceId' => 7])->call('refreshOverview');
     */
    public function refreshOverview(): void
    {
        unset($this->nodes, $this->stats, $this->platforms);
        $this->syncDispatchAgentId();
        $this->dispatch('notify', message: 'Fleet overview refreshed');
    }

    /**
     * Map a fleet node status to the badge variant used in the UI.
     *
     * @example
     * $variant = $component->statusBadgeVariant(FleetNode::STATUS_BUSY);
     */
    public function statusBadgeVariant(string $status): string
    {
        return match ($status) {
            FleetNode::STATUS_BUSY => 'warning',
            FleetNode::STATUS_ONLINE => 'success',
            FleetNode::STATUS_PAUSED => 'zinc',
            default => 'danger',
        };
    }

    /**
     * Keep the dispatch agent aligned with the currently visible node list.
     *
     * @example
     * $this->syncDispatchAgentId();
     */
    private function syncDispatchAgentId(): void
    {
        if ($this->dispatchAgentId !== '' && collect($this->nodes)->contains('agent_id', $this->dispatchAgentId)) {
            return;
        }

        $this->dispatchAgentId = (string) (collect($this->nodes)->first()['agent_id'] ?? '');
    }

    /**
     * @param  array<string, mixed>  $budget
     */
    private function summariseBudget(array $budget): string
    {
        if ($budget === []) {
            return 'Not set';
        }

        return collect($budget)
            ->map(
                static fn (mixed $value, string|int $key): string => sprintf(
                    '%s: %s',
                    (string) $key,
                    is_scalar($value) ? (string) $value : json_encode($value),
                ),
            )
            ->implode(', ');
    }

    /**
     * Resolve the Blade template used by the fleet overview screen.
     *
     * @example
     * $path = $this->viewPath();
     */
    protected function viewPath(): string
    {
        return __DIR__.'/../resources/views/livewire/agentic/fleet-overview.blade.php';
    }
}
