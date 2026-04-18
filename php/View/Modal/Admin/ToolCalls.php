<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mcp\Models\McpToolCall;
use Core\Mcp\Models\McpToolCallStat;
use Core\Tenant\Models\Workspace;
use Illuminate\Contracts\Pagination\LengthAwarePaginator;
use Illuminate\Contracts\View\View;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithPagination;

#[Title('Tool Calls')]
#[Layout('hub::admin.layouts.app')]
class ToolCalls extends Component
{
    use WithPagination;

    #[Url]
    public string $search = '';

    #[Url]
    public string $server = '';

    #[Url]
    public string $tool = '';

    #[Url]
    public string $status = '';

    #[Url]
    public string $workspace = '';

    #[Url]
    public string $agentType = '';

    public int $perPage = 25;

    public ?int $selectedCallId = null;

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function calls(): LengthAwarePaginator
    {
        $query = McpToolCall::query()
            ->with('workspace')
            ->orderByDesc('created_at');

        if ($this->search) {
            $query->where(function ($q) {
                $q->where('tool_name', 'like', "%{$this->search}%")
                    ->orWhere('server_id', 'like', "%{$this->search}%")
                    ->orWhere('session_id', 'like', "%{$this->search}%")
                    ->orWhere('error_message', 'like', "%{$this->search}%");
            });
        }

        if ($this->server) {
            $query->forServer($this->server);
        }

        if ($this->tool) {
            $query->forTool($this->tool);
        }

        if ($this->status === 'success') {
            $query->successful();
        } elseif ($this->status === 'failed') {
            $query->failed();
        }

        if ($this->workspace) {
            $query->where('workspace_id', $this->workspace);
        }

        if ($this->agentType) {
            $query->where('agent_type', $this->agentType);
        }

        return $query->paginate($this->perPage);
    }

    #[Computed]
    public function workspaces(): Collection
    {
        return Workspace::orderBy('name')->get();
    }

    #[Computed]
    public function servers(): Collection
    {
        return McpToolCallStat::query()
            ->select('server_id')
            ->distinct()
            ->orderBy('server_id')
            ->pluck('server_id');
    }

    #[Computed]
    public function tools(): Collection
    {
        $query = McpToolCallStat::query()
            ->select('tool_name')
            ->distinct()
            ->orderBy('tool_name');

        if ($this->server) {
            $query->where('server_id', $this->server);
        }

        return $query->pluck('tool_name');
    }

    #[Computed]
    public function agentTypes(): array
    {
        return [
            'opus' => 'Opus',
            'sonnet' => 'Sonnet',
            'haiku' => 'Haiku',
        ];
    }

    #[Computed]
    public function selectedCall(): ?McpToolCall
    {
        if (! $this->selectedCallId) {
            return null;
        }

        return McpToolCall::with('workspace')->find($this->selectedCallId);
    }

    public function viewCall(int $id): void
    {
        $this->selectedCallId = $id;
    }

    public function closeCallDetail(): void
    {
        $this->selectedCallId = null;
    }

    public function clearFilters(): void
    {
        $this->search = '';
        $this->server = '';
        $this->tool = '';
        $this->status = '';
        $this->workspace = '';
        $this->agentType = '';
        $this->resetPage();
    }

    public function getStatusBadgeClass(bool $success): string
    {
        return $success
            ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300'
            : 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300';
    }

    public function getAgentBadgeClass(?string $agentType): string
    {
        return match ($agentType) {
            'opus' => 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300',
            'sonnet' => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            'haiku' => 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/50 dark:text-cyan-300',
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
        return view('agentic::admin.tool-calls');
    }
}
