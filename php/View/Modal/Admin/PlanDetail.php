<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Illuminate\Contracts\View\View;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Plan Detail')]
#[Layout('hub::admin.layouts.app')]
class PlanDetail extends Component
{
    public AgentPlan $plan;

    public bool $showAddTaskModal = false;

    public int $selectedPhaseId = 0;

    public string $newTaskName = '';

    public string $newTaskNotes = '';

    public function mount(string $slug): void
    {
        $this->checkHadesAccess();

        $this->plan = AgentPlan::where('slug', $slug)
            ->with(['workspace', 'agentPhases', 'sessions'])
            ->firstOrFail();
    }

    #[Computed]
    public function progress(): array
    {
        return $this->plan->getProgress();
    }

    #[Computed]
    public function phases(): \Illuminate\Database\Eloquent\Collection
    {
        return $this->plan->agentPhases()->orderBy('order')->get();
    }

    #[Computed]
    public function sessions(): \Illuminate\Database\Eloquent\Collection
    {
        return $this->plan->sessions()->latest('started_at')->get();
    }

    // Plan status actions
    public function activatePlan(): void
    {
        $this->plan->activate();
        $this->dispatch('notify', message: 'Plan activated');
    }

    public function completePlan(): void
    {
        $this->plan->complete();
        $this->dispatch('notify', message: 'Plan completed');
    }

    public function archivePlan(): void
    {
        $this->plan->archive('Archived via admin UI');
        $this->dispatch('notify', message: 'Plan archived');
        $this->redirect(route('hub.agents.plans'), navigate: true);
    }

    // Phase status actions
    public function startPhase(int $phaseId): void
    {
        $phase = AgentPhase::findOrFail($phaseId);

        if (! $phase->canStart()) {
            $this->dispatch('notify', message: 'Phase cannot start - dependencies not met', type: 'error');

            return;
        }

        $phase->start();
        $this->plan->refresh();
        $this->dispatch('notify', message: "Phase \"{$phase->name}\" started");
    }

    public function completePhase(int $phaseId): void
    {
        $phase = AgentPhase::findOrFail($phaseId);
        $phase->complete();
        $this->plan->refresh();
        $this->dispatch('notify', message: "Phase \"{$phase->name}\" completed");
    }

    public function blockPhase(int $phaseId): void
    {
        $phase = AgentPhase::findOrFail($phaseId);
        $phase->block('Blocked via admin UI');
        $this->plan->refresh();
        $this->dispatch('notify', message: "Phase \"{$phase->name}\" blocked");
    }

    public function skipPhase(int $phaseId): void
    {
        $phase = AgentPhase::findOrFail($phaseId);
        $phase->skip('Skipped via admin UI');
        $this->plan->refresh();
        $this->dispatch('notify', message: "Phase \"{$phase->name}\" skipped");
    }

    public function resetPhase(int $phaseId): void
    {
        $phase = AgentPhase::findOrFail($phaseId);
        $phase->reset();
        $this->plan->refresh();
        $this->dispatch('notify', message: "Phase \"{$phase->name}\" reset to pending");
    }

    // Task management
    public function completeTask(int $phaseId, string|int $taskIdentifier): void
    {
        $phase = AgentPhase::findOrFail($phaseId);
        $phase->completeTask($taskIdentifier);
        $this->plan->refresh();
        $this->dispatch('notify', message: 'Task completed');
    }

    public function openAddTaskModal(int $phaseId): void
    {
        $this->selectedPhaseId = $phaseId;
        $this->newTaskName = '';
        $this->newTaskNotes = '';
        $this->showAddTaskModal = true;
    }

    public function addTask(): void
    {
        $this->validate([
            'newTaskName' => 'required|string|max:255',
            'newTaskNotes' => 'nullable|string|max:1000',
        ]);

        $phase = AgentPhase::findOrFail($this->selectedPhaseId);
        $phase->addTask($this->newTaskName, $this->newTaskNotes ?: null);

        $this->showAddTaskModal = false;
        $this->newTaskName = '';
        $this->newTaskNotes = '';
        $this->plan->refresh();

        $this->dispatch('notify', message: 'Task added');
    }

    public function getStatusColorClass(string $status): string
    {
        return match ($status) {
            AgentPlan::STATUS_DRAFT => 'bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
            AgentPlan::STATUS_ACTIVE => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            AgentPlan::STATUS_COMPLETED => 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300',
            AgentPlan::STATUS_ARCHIVED => 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
            AgentPhase::STATUS_PENDING => 'bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
            AgentPhase::STATUS_IN_PROGRESS => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            AgentPhase::STATUS_COMPLETED => 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300',
            AgentPhase::STATUS_BLOCKED => 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
            AgentPhase::STATUS_SKIPPED => 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
            default => 'bg-zinc-100 text-zinc-700',
        };
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render(): View
    {
        return view('agentic::admin.plan-detail');
    }
}
