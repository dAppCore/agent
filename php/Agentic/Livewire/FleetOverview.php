<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Livewire;

use Core\Mod\Agentic\Actions\Fleet\AssignTask;
use Core\Mod\Agentic\Actions\Fleet\GetFleetStats;
use Core\Mod\Agentic\Actions\Fleet\ListNodes;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Tenant\Models\Workspace;
use Flux\Flux;
use Illuminate\Contracts\View\View;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Fleet Overview')]
#[Layout('hub::admin.layouts.app')]
class FleetOverview extends Component
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

    public function mount(?int $workspaceId = null): void
    {
        $this->checkHadesAccess();
        $this->workspaceId = $workspaceId ?? $this->resolveWorkspaceId();
        $this->syncDispatchAgentId();
    }

    #[Computed]
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
                        'current_task_label' => $currentTask?->repo ?? ($node->current_task_id ? 'Task #'.$node->current_task_id : 'Idle'),
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

    public function stageDispatch(string $agentId): void
    {
        $this->dispatchAgentId = $agentId;
        $this->toast('Dispatch Ready', "Prepared dispatch for {$agentId}.", 'info');
    }

    public function dispatchTask(): void
    {
        $this->validate([
            'workspaceId' => 'required|integer|min:1',
            'dispatchAgentId' => 'required|string|max:255',
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

    public function refreshOverview(): void
    {
        unset($this->nodes, $this->stats, $this->platforms);
        $this->syncDispatchAgentId();
        $this->dispatch('notify', message: 'Fleet overview refreshed');
    }

    public function statusBadgeVariant(string $status): string
    {
        return match ($status) {
            FleetNode::STATUS_BUSY => 'warning',
            FleetNode::STATUS_ONLINE => 'success',
            FleetNode::STATUS_PAUSED => 'zinc',
            default => 'danger',
        };
    }

    public function render(): View
    {
        return view()->file($this->viewPath());
    }

    private function resolveWorkspaceId(): int
    {
        try {
            return (int) (Workspace::query()->value('id') ?? 0);
        } catch (\Throwable) {
            return 0;
        }
    }

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
            ->map(static fn (mixed $value, string|int $key): string => sprintf('%s: %s', (string) $key, is_scalar($value) ? (string) $value : json_encode($value)))
            ->implode(', ');
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    private function toast(string $heading, string $text, string $variant): void
    {
        if (class_exists(Flux::class)) {
            Flux::toast(
                heading: $heading,
                text: $text,
                variant: $variant,
            );

            return;
        }

        $this->dispatch('notify', message: $text, variant: $variant);
    }

    private function viewPath(): string
    {
        return __DIR__.'/../../resources/views/livewire/agentic/fleet-overview.blade.php';
    }
}
