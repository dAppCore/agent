<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mcp\Models\McpToolCallStat;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Illuminate\Cache\Lock;
use Illuminate\Contracts\View\View;
use Illuminate\Support\Facades\Cache;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Agent Operations')]
#[Layout('hub::admin.layouts.app')]
class Dashboard extends Component
{
    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function stats(): array
    {
        return $this->cacheWithLock('admin.agents.dashboard.stats', 60, function () {
            try {
                $activePlans = AgentPlan::active()->count();
                $totalPlans = AgentPlan::notArchived()->count();
            } catch (\Throwable) {
                $activePlans = 0;
                $totalPlans = 0;
            }

            try {
                $activeSessions = AgentSession::active()->count();
                $todaySessions = AgentSession::whereDate('started_at', today())->count();
            } catch (\Throwable) {
                $activeSessions = 0;
                $todaySessions = 0;
            }

            try {
                $toolStats = McpToolCallStat::last7Days()
                    ->selectRaw('SUM(call_count) as total_calls')
                    ->selectRaw('SUM(success_count) as total_success')
                    ->first();
                $totalCalls = $toolStats->total_calls ?? 0;
                $totalSuccess = $toolStats->total_success ?? 0;
            } catch (\Throwable) {
                $totalCalls = 0;
                $totalSuccess = 0;
            }

            $successRate = $totalCalls > 0 ? round(($totalSuccess / $totalCalls) * 100, 1) : 0;

            return [
                'active_plans' => $activePlans,
                'total_plans' => $totalPlans,
                'active_sessions' => $activeSessions,
                'today_sessions' => $todaySessions,
                'tool_calls_7d' => $totalCalls,
                'success_rate' => $successRate,
            ];
        });
    }

    #[Computed]
    public function statCards(): array
    {
        $rate = $this->stats['success_rate'];
        $rateColor = $rate >= 95 ? 'green' : ($rate >= 80 ? 'amber' : 'red');

        return [
            ['value' => $this->stats['active_plans'], 'label' => 'Active Plans', 'icon' => 'clipboard-document-list', 'color' => 'blue'],
            ['value' => $this->stats['active_sessions'], 'label' => 'Active Sessions', 'icon' => 'play', 'color' => 'green'],
            ['value' => number_format($this->stats['tool_calls_7d']), 'label' => 'Tool Calls (7d)', 'icon' => 'wrench', 'color' => 'violet'],
            ['value' => $this->stats['success_rate'].'%', 'label' => 'Success Rate', 'icon' => 'check-circle', 'color' => $rateColor],
        ];
    }

    #[Computed]
    public function blockedAlert(): ?array
    {
        if ($this->blockedPlans === 0) {
            return null;
        }

        return [
            'type' => 'warning',
            'title' => $this->blockedPlans.' plan(s) have blocked phases',
            'message' => 'Review and unblock to continue agent work',
            'action' => ['label' => 'View Plans', 'href' => route('hub.agents.plans', ['status' => 'active'])],
        ];
    }

    #[Computed]
    public function activityItems(): array
    {
        return collect($this->recentActivity)->map(fn ($a) => [
            'message' => $a['title'],
            'subtitle' => $a['workspace'].' - '.$a['description'],
            'time' => $a['time']->diffForHumans(),
            'icon' => $a['icon'],
            'color' => $a['type'] === 'plan' ? 'blue' : 'green',
        ])->all();
    }

    #[Computed]
    public function toolItems(): array
    {
        return $this->topTools->map(fn ($t) => [
            'label' => $t->tool_name,
            'value' => $t->total_calls,
            'subtitle' => $t->server_id,
            'badge' => $t->success_rate.'% success',
            'badgeColor' => $t->success_rate >= 95 ? 'green' : ($t->success_rate >= 80 ? 'amber' : 'red'),
        ])->all();
    }

    #[Computed]
    public function quickLinks(): array
    {
        return [
            ['href' => route('hub.agents.plans'), 'label' => 'All Plans', 'icon' => 'clipboard-document-list', 'color' => 'blue'],
            ['href' => route('hub.agents.sessions'), 'label' => 'Sessions', 'icon' => 'play', 'color' => 'green'],
            ['href' => route('hub.agents.tools'), 'label' => 'Tool Analytics', 'icon' => 'chart-bar', 'color' => 'violet'],
            ['href' => route('hub.agents.templates'), 'label' => 'Templates', 'icon' => 'document-duplicate', 'color' => 'amber'],
        ];
    }

    #[Computed]
    public function recentActivity(): array
    {
        return $this->cacheWithLock('admin.agents.dashboard.activity', 30, function () {
            $activities = [];

            try {
                $plans = AgentPlan::with('workspace')
                    ->latest('updated_at')
                    ->take(5)
                    ->get();

                foreach ($plans as $plan) {
                    $activities[] = [
                        'type' => 'plan',
                        'icon' => 'clipboard-list',
                        'title' => "Plan \"{$plan->title}\"",
                        'description' => "Status: {$plan->status}",
                        'workspace' => $plan->workspace?->name ?? 'Unknown',
                        'time' => $plan->updated_at,
                        'link' => route('hub.agents.plans.show', $plan->slug),
                    ];
                }
            } catch (\Throwable) {
                // Table may not exist yet
            }

            try {
                $sessions = AgentSession::with(['plan', 'workspace'])
                    ->latest('last_active_at')
                    ->take(5)
                    ->get();

                foreach ($sessions as $session) {
                    $activities[] = [
                        'type' => 'session',
                        'icon' => 'play',
                        'title' => "Session {$session->session_id}",
                        'description' => $session->plan?->title ?? 'No plan',
                        'workspace' => $session->workspace?->name ?? 'Unknown',
                        'time' => $session->last_active_at ?? $session->created_at,
                        'link' => route('hub.agents.sessions.show', $session->id),
                    ];
                }
            } catch (\Throwable) {
                // Table may not exist yet
            }

            // Sort by time descending
            usort($activities, fn ($a, $b) => $b['time'] <=> $a['time']);

            return array_slice($activities, 0, 10);
        });
    }

    #[Computed]
    public function topTools(): \Illuminate\Support\Collection
    {
        return $this->cacheWithLock('admin.agents.dashboard.toptools', 300, function () {
            try {
                return McpToolCallStat::getTopTools(days: 7, limit: 5);
            } catch (\Throwable) {
                return collect();
            }
        });
    }

    #[Computed]
    public function dailyTrend(): \Illuminate\Support\Collection
    {
        return $this->cacheWithLock('admin.agents.dashboard.dailytrend', 300, function () {
            try {
                return McpToolCallStat::getDailyTrend(days: 7);
            } catch (\Throwable) {
                return collect();
            }
        });
    }

    #[Computed]
    public function blockedPlans(): int
    {
        return (int) $this->cacheWithLock('admin.agents.dashboard.blocked', 60, function () {
            try {
                return AgentPlan::active()
                    ->whereHas('agentPhases', function ($query) {
                        $query->where('status', 'blocked');
                    })
                    ->count();
            } catch (\Throwable) {
                return 0;
            }
        });
    }

    /**
     * Cache with lock to prevent cache stampede.
     *
     * Uses atomic locks to ensure only one request regenerates cache while
     * others return stale data or wait briefly.
     */
    private function cacheWithLock(string $key, int $ttl, callable $callback): mixed
    {
        // Try to get from cache first
        $value = Cache::get($key);

        if ($value !== null) {
            return $value;
        }

        // Try to acquire lock for regeneration (wait up to 5 seconds)
        $lock = Cache::lock($key.':lock', 10);

        if ($lock->get()) {
            try {
                // Double-check cache after acquiring lock
                $value = Cache::get($key);
                if ($value !== null) {
                    return $value;
                }

                // Generate and cache the value
                $value = $callback();
                Cache::put($key, $value, $ttl);

                return $value;
            } finally {
                $lock->release();
            }
        }

        // Could not acquire lock, return default/empty value
        // This prevents blocking when another request is regenerating
        return $callback();
    }

    public function refresh(): void
    {
        Cache::forget('admin.agents.dashboard.stats');
        Cache::forget('admin.agents.dashboard.activity');
        Cache::forget('admin.agents.dashboard.toptools');
        Cache::forget('admin.agents.dashboard.dailytrend');
        Cache::forget('admin.agents.dashboard.blocked');

        unset($this->stats);
        unset($this->recentActivity);
        unset($this->topTools);
        unset($this->dailyTrend);
        unset($this->blockedPlans);

        $this->dispatch('notify', message: 'Dashboard refreshed');
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render(): View
    {
        return view('agentic::admin.dashboard');
    }
}
