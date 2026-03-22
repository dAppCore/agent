<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;
use Illuminate\Contracts\Pagination\LengthAwarePaginator;
use Illuminate\Contracts\View\View;
use Illuminate\Database\Eloquent\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithPagination;

#[Title('Agent Sessions')]
#[Layout('hub::admin.layouts.app')]
class Sessions extends Component
{
    use WithPagination;

    #[Url]
    public string $search = '';

    #[Url]
    public string $status = '';

    #[Url]
    public string $agentType = '';

    #[Url]
    public string $workspace = '';

    #[Url]
    public string $planSlug = '';

    public int $perPage = 20;

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function sessions(): LengthAwarePaginator
    {
        $query = AgentSession::with(['workspace', 'plan']);

        if ($this->search) {
            $query->where(function ($q) {
                $q->where('session_id', 'like', "%{$this->search}%")
                    ->orWhere('agent_type', 'like', "%{$this->search}%")
                    ->orWhereHas('plan', fn ($p) => $p->where('title', 'like', "%{$this->search}%"));
            });
        }

        if ($this->status) {
            $query->where('status', $this->status);
        }

        if ($this->agentType) {
            $query->where('agent_type', $this->agentType);
        }

        if ($this->workspace) {
            $query->where('workspace_id', $this->workspace);
        }

        if ($this->planSlug) {
            $query->whereHas('plan', fn ($q) => $q->where('slug', $this->planSlug));
        }

        return $query->latest('last_active_at')->paginate($this->perPage);
    }

    #[Computed]
    public function statusOptions(): array
    {
        return [
            AgentSession::STATUS_ACTIVE => 'Active',
            AgentSession::STATUS_PAUSED => 'Paused',
            AgentSession::STATUS_COMPLETED => 'Completed',
            AgentSession::STATUS_FAILED => 'Failed',
        ];
    }

    #[Computed]
    public function agentTypes(): array
    {
        return [
            AgentSession::AGENT_OPUS => 'Opus',
            AgentSession::AGENT_SONNET => 'Sonnet',
            AgentSession::AGENT_HAIKU => 'Haiku',
        ];
    }

    #[Computed]
    public function workspaces(): Collection
    {
        return Workspace::orderBy('name')->get();
    }

    #[Computed]
    public function plans(): Collection
    {
        return AgentPlan::orderBy('title')->get(['id', 'title', 'slug']);
    }

    #[Computed]
    public function activeCount(): int
    {
        return AgentSession::active()->count();
    }

    public function clearFilters(): void
    {
        $this->search = '';
        $this->status = '';
        $this->agentType = '';
        $this->workspace = '';
        $this->planSlug = '';
        $this->resetPage();
    }

    public function pause(int $sessionId): void
    {
        $session = AgentSession::findOrFail($sessionId);
        $session->pause();
        $this->dispatch('notify', message: 'Session paused');
    }

    public function resume(int $sessionId): void
    {
        $session = AgentSession::findOrFail($sessionId);
        $session->resume();
        $this->dispatch('notify', message: 'Session resumed');
    }

    public function complete(int $sessionId): void
    {
        $session = AgentSession::findOrFail($sessionId);
        $session->complete('Completed via admin UI');
        $this->dispatch('notify', message: 'Session completed');
    }

    public function fail(int $sessionId): void
    {
        $session = AgentSession::findOrFail($sessionId);
        $session->fail('Failed via admin UI');
        $this->dispatch('notify', message: 'Session marked as failed');
    }

    public function getStatusColorClass(string $status): string
    {
        return match ($status) {
            AgentSession::STATUS_ACTIVE => 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300',
            AgentSession::STATUS_PAUSED => 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
            AgentSession::STATUS_COMPLETED => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            AgentSession::STATUS_FAILED => 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
            default => 'bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
        };
    }

    public function getAgentBadgeClass(string $agentType): string
    {
        return match ($agentType) {
            AgentSession::AGENT_OPUS => 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300',
            AgentSession::AGENT_SONNET => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            AgentSession::AGENT_HAIKU => 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/50 dark:text-cyan-300',
            default => 'bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
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
        return view('agentic::admin.sessions');
    }
}
