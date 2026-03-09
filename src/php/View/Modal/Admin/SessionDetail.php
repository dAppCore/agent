<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Models\AgentSession;
use Illuminate\Contracts\View\View;
use Illuminate\Database\Eloquent\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Session Detail')]
#[Layout('hub::admin.layouts.app')]
class SessionDetail extends Component
{
    public AgentSession $session;

    public bool $showCompleteModal = false;

    public bool $showFailModal = false;

    public bool $showReplayModal = false;

    public string $completeSummary = '';

    public string $failReason = '';

    public string $replayAgentType = '';

    // Polling interval in milliseconds (5 seconds for active sessions)
    public int $pollingInterval = 5000;

    public function mount(int $id): void
    {
        $this->checkHadesAccess();

        $this->session = AgentSession::with(['workspace', 'plan', 'plan.agentPhases'])
            ->findOrFail($id);

        // Disable polling for completed/failed sessions
        if ($this->session->isEnded()) {
            $this->pollingInterval = 0;
        }
    }

    #[Computed]
    public function workLog(): array
    {
        return $this->session->work_log ?? [];
    }

    #[Computed]
    public function recentWorkLog(): array
    {
        $log = $this->session->work_log ?? [];

        return array_reverse($log);
    }

    #[Computed]
    public function artifacts(): array
    {
        return $this->session->artifacts ?? [];
    }

    #[Computed]
    public function handoffNotes(): ?array
    {
        return $this->session->handoff_notes;
    }

    #[Computed]
    public function contextSummary(): ?array
    {
        return $this->session->context_summary;
    }

    #[Computed]
    public function planSessions(): Collection
    {
        if (! $this->session->agent_plan_id) {
            return collect();
        }

        return AgentSession::where('agent_plan_id', $this->session->agent_plan_id)
            ->orderBy('started_at')
            ->get();
    }

    #[Computed]
    public function sessionIndex(): int
    {
        if (! $this->session->agent_plan_id) {
            return 0;
        }

        $sessions = $this->planSessions;
        foreach ($sessions as $index => $s) {
            if ($s->id === $this->session->id) {
                return $index + 1;
            }
        }

        return 0;
    }

    // Polling method for real-time updates
    public function poll(): void
    {
        // Refresh session data
        $this->session->refresh();

        // Disable polling if session ended
        if ($this->session->isEnded()) {
            $this->pollingInterval = 0;
        }
    }

    // Session actions
    public function pauseSession(): void
    {
        $this->session->pause();
        $this->dispatch('notify', message: 'Session paused');
    }

    public function resumeSession(): void
    {
        $this->session->resume();
        $this->pollingInterval = 5000; // Re-enable polling
        $this->dispatch('notify', message: 'Session resumed');
    }

    public function openCompleteModal(): void
    {
        $this->completeSummary = '';
        $this->showCompleteModal = true;
    }

    public function completeSession(): void
    {
        $this->session->complete($this->completeSummary ?: 'Completed via admin UI');
        $this->showCompleteModal = false;
        $this->pollingInterval = 0;
        $this->dispatch('notify', message: 'Session completed');
    }

    public function openFailModal(): void
    {
        $this->failReason = '';
        $this->showFailModal = true;
    }

    public function failSession(): void
    {
        $this->session->fail($this->failReason ?: 'Failed via admin UI');
        $this->showFailModal = false;
        $this->pollingInterval = 0;
        $this->dispatch('notify', message: 'Session marked as failed');
    }

    public function openReplayModal(): void
    {
        $this->replayAgentType = $this->session->agent_type ?? '';
        $this->showReplayModal = true;
    }

    public function replaySession(): void
    {
        $newSession = $this->session->createReplaySession(
            $this->replayAgentType ?: null
        );

        $this->showReplayModal = false;
        $this->dispatch('notify', message: 'Session replayed successfully');

        // Redirect to the new session
        $this->redirect(route('hub.agents.sessions.show', $newSession->id), navigate: true);
    }

    #[Computed]
    public function replayContext(): array
    {
        return $this->session->getReplayContext();
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

    public function getAgentBadgeClass(?string $agentType): string
    {
        return match ($agentType) {
            AgentSession::AGENT_OPUS => 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300',
            AgentSession::AGENT_SONNET => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            AgentSession::AGENT_HAIKU => 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/50 dark:text-cyan-300',
            default => 'bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
        };
    }

    public function getLogTypeIcon(string $type): string
    {
        return match ($type) {
            'success' => 'check-circle',
            'error' => 'x-circle',
            'warning' => 'exclamation-triangle',
            'checkpoint' => 'flag',
            default => 'information-circle',
        };
    }

    public function getLogTypeColor(string $type): string
    {
        return match ($type) {
            'success' => 'text-green-500',
            'error' => 'text-red-500',
            'warning' => 'text-amber-500',
            'checkpoint' => 'text-violet-500',
            default => 'text-blue-500',
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
        return view('agentic::admin.session-detail');
    }
}
