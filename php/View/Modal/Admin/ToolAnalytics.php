<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mcp\Models\McpToolCall;
use Core\Mcp\Models\McpToolCallStat;
use Core\Tenant\Models\Workspace;
use Illuminate\Contracts\View\View;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;

#[Title('Tool Analytics')]
#[Layout('hub::admin.layouts.app')]
class ToolAnalytics extends Component
{
    #[Url]
    public int $days = 7;

    #[Url]
    public string $workspace = '';

    #[Url]
    public string $server = '';

    public function mount(): void
    {
        $this->checkHadesAccess();
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
    public function stats(): array
    {
        $workspaceId = $this->workspace ? (int) $this->workspace : null;

        $topTools = McpToolCallStat::getTopTools($this->days, 100, $workspaceId);

        // Filter by server if selected
        if ($this->server) {
            $topTools = $topTools->filter(fn ($t) => $t->server_id === $this->server);
        }

        $totalCalls = $topTools->sum('total_calls');
        $totalSuccess = $topTools->sum('total_success');
        $totalErrors = $topTools->sum('total_errors');
        $successRate = $totalCalls > 0 ? round(($totalSuccess / $totalCalls) * 100, 1) : 0;
        $uniqueTools = $topTools->count();

        return [
            'total_calls' => $totalCalls,
            'total_success' => $totalSuccess,
            'total_errors' => $totalErrors,
            'success_rate' => $successRate,
            'unique_tools' => $uniqueTools,
        ];
    }

    #[Computed]
    public function topTools(): Collection
    {
        $workspaceId = $this->workspace ? (int) $this->workspace : null;
        $tools = McpToolCallStat::getTopTools($this->days, 10, $workspaceId);

        if ($this->server) {
            $tools = $tools->filter(fn ($t) => $t->server_id === $this->server)->values();
        }

        return $tools->take(10);
    }

    #[Computed]
    public function dailyTrend(): Collection
    {
        $workspaceId = $this->workspace ? (int) $this->workspace : null;

        return McpToolCallStat::getDailyTrend($this->days, $workspaceId);
    }

    #[Computed]
    public function serverStats(): Collection
    {
        $workspaceId = $this->workspace ? (int) $this->workspace : null;
        $stats = McpToolCallStat::getServerStats($this->days, $workspaceId);

        if ($this->server) {
            $stats = $stats->filter(fn ($s) => $s->server_id === $this->server)->values();
        }

        return $stats;
    }

    #[Computed]
    public function recentErrors(): Collection
    {
        $query = McpToolCall::query()
            ->failed()
            ->with('workspace')
            ->orderByDesc('created_at')
            ->limit(10);

        if ($this->workspace) {
            $query->where('workspace_id', $this->workspace);
        }

        if ($this->server) {
            $query->forServer($this->server);
        }

        return $query->get();
    }

    #[Computed]
    public function chartData(): array
    {
        $trend = $this->dailyTrend;

        return [
            'labels' => $trend->pluck('date')->map(fn ($d) => $d->format('M j'))->toArray(),
            'calls' => $trend->pluck('total_calls')->toArray(),
            'errors' => $trend->pluck('total_errors')->toArray(),
            'success_rates' => $trend->pluck('success_rate')->toArray(),
        ];
    }

    public function clearFilters(): void
    {
        $this->workspace = '';
        $this->server = '';
        $this->days = 7;
    }

    public function setDays(int $days): void
    {
        $this->days = $days;
    }

    public function getSuccessRateColorClass(float $rate): string
    {
        return match (true) {
            $rate >= 95 => 'text-green-500',
            $rate >= 80 => 'text-amber-500',
            default => 'text-red-500',
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
        return view('agentic::admin.tool-analytics');
    }
}
