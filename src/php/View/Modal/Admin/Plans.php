<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\Workspace;
use Illuminate\Contracts\View\View;
use Illuminate\Pagination\LengthAwarePaginator;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithPagination;

#[Title('Agent Plans')]
#[Layout('hub::admin.layouts.app')]
class Plans extends Component
{
    use WithPagination;

    #[Url]
    public string $search = '';

    #[Url]
    public string $status = '';

    #[Url]
    public string $workspace = '';

    public int $perPage = 15;

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function plans(): LengthAwarePaginator
    {
        $query = AgentPlan::with(['workspace', 'agentPhases'])
            ->withCount('sessions');

        if ($this->search) {
            $query->where(function ($q) {
                $q->where('title', 'like', "%{$this->search}%")
                    ->orWhere('slug', 'like', "%{$this->search}%")
                    ->orWhere('description', 'like', "%{$this->search}%");
            });
        }

        if ($this->status) {
            $query->where('status', $this->status);
        }

        if ($this->workspace) {
            $query->where('workspace_id', $this->workspace);
        }

        return $query->latest('updated_at')->paginate($this->perPage);
    }

    #[Computed]
    public function workspaces(): \Illuminate\Database\Eloquent\Collection
    {
        return Workspace::orderBy('name')->get();
    }

    #[Computed]
    public function statusOptions(): array
    {
        return [
            AgentPlan::STATUS_DRAFT => 'Draft',
            AgentPlan::STATUS_ACTIVE => 'Active',
            AgentPlan::STATUS_COMPLETED => 'Completed',
            AgentPlan::STATUS_ARCHIVED => 'Archived',
        ];
    }

    public function updatedSearch(): void
    {
        $this->resetPage();
    }

    public function updatedStatus(): void
    {
        $this->resetPage();
    }

    public function updatedWorkspace(): void
    {
        $this->resetPage();
    }

    public function clearFilters(): void
    {
        $this->search = '';
        $this->status = '';
        $this->workspace = '';
        $this->resetPage();
    }

    public function activate(int $planId): void
    {
        $plan = AgentPlan::findOrFail($planId);
        $plan->activate();
        $this->dispatch('notify', message: "Plan \"{$plan->title}\" activated");
    }

    public function complete(int $planId): void
    {
        $plan = AgentPlan::findOrFail($planId);
        $plan->complete();
        $this->dispatch('notify', message: "Plan \"{$plan->title}\" marked complete");
    }

    public function archive(int $planId): void
    {
        $plan = AgentPlan::findOrFail($planId);
        $plan->archive('Archived via admin UI');
        $this->dispatch('notify', message: "Plan \"{$plan->title}\" archived");
    }

    public function delete(int $planId): void
    {
        $plan = AgentPlan::findOrFail($planId);
        $title = $plan->title;
        $plan->delete();
        $this->dispatch('notify', message: "Plan \"{$title}\" deleted");
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render(): View
    {
        return view('agentic::admin.plans');
    }
}
