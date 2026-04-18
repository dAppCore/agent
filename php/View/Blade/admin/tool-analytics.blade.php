<div>
    {{-- Header --}}
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
        <div>
            <core:heading size="xl">{{ __('agentic::agentic.tools.title') }}</core:heading>
            <core:subheading>{{ __('agentic::agentic.tools.subtitle') }}</core:subheading>
        </div>
        <div class="flex items-center gap-2">
            <a href="{{ route('hub.agents.tools.calls') }}" wire:navigate>
                <core:button variant="ghost" icon="list-bullet">{{ __('agentic::agentic.actions.view_all_calls') }}</core:button>
            </a>
        </div>
    </div>

    {{-- Filters --}}
    <core:card class="p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-4">
            <div class="w-full md:w-48">
                <core:select wire:model.live="days">
                    <option value="7">{{ __('agentic::agentic.filters.last_7_days') }}</option>
                    <option value="14">{{ __('agentic::agentic.filters.last_14_days') }}</option>
                    <option value="30">{{ __('agentic::agentic.filters.last_30_days') }}</option>
                    <option value="90">{{ __('agentic::agentic.filters.last_90_days') }}</option>
                </core:select>
            </div>
            <div class="w-full md:w-48">
                <core:select wire:model.live="workspace">
                    <option value="">{{ __('agentic::agentic.filters.all_workspaces') }}</option>
                    @foreach($this->workspaces as $ws)
                        <option value="{{ $ws->id }}">{{ $ws->name }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-48">
                <core:select wire:model.live="server">
                    <option value="">{{ __('agentic::agentic.filters.all_servers') }}</option>
                    @foreach($this->servers as $srv)
                        <option value="{{ $srv }}">{{ $srv }}</option>
                    @endforeach
                </core:select>
            </div>
            @if($workspace || $server || $days !== 7)
                <core:button wire:click="clearFilters" variant="ghost" icon="x-mark">
                    {{ __('agentic::agentic.actions.clear') }}
                </core:button>
            @endif
        </div>
    </core:card>

    {{-- Stats Cards --}}
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
                    <core:icon name="wrench" class="w-5 h-5 text-blue-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.stats.total_calls') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ number_format($this->stats['total_calls']) }}</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                    <core:icon name="check-circle" class="w-5 h-5 text-green-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.stats.successful') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ number_format($this->stats['total_success']) }}</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center">
                    <core:icon name="x-circle" class="w-5 h-5 text-red-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.stats.errors') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ number_format($this->stats['total_errors']) }}</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-{{ $this->stats['success_rate'] >= 95 ? 'green' : ($this->stats['success_rate'] >= 80 ? 'amber' : 'red') }}-500/10 flex items-center justify-center">
                    <core:icon name="chart-bar" class="w-5 h-5 text-{{ $this->stats['success_rate'] >= 95 ? 'green' : ($this->stats['success_rate'] >= 80 ? 'amber' : 'red') }}-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.stats.success_rate') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ $this->stats['success_rate'] }}%</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-violet-500/10 flex items-center justify-center">
                    <core:icon name="squares-2x2" class="w-5 h-5 text-violet-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.stats.unique_tools') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ $this->stats['unique_tools'] }}</core:text>
                </div>
            </div>
        </core:card>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {{-- Daily Trend Chart (AC15) --}}
        <core:card class="p-6">
            <div class="flex items-center justify-between mb-4">
                <core:heading size="lg">{{ __('agentic::agentic.tools.daily_trend') }}</core:heading>
                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.day_window', ['days' => $days]) }}</core:text>
            </div>
            @if($this->dailyTrend->count() > 0)
                <div class="h-64" x-data="chartComponent(@js($this->chartData))" x-init="initChart()">
                    <canvas x-ref="chart"></canvas>
                </div>
            @else
                <div class="flex flex-col items-center justify-center h-64">
                    <core:icon name="chart-bar" class="w-12 h-12 text-zinc-300 dark:text-zinc-600 mb-3" />
                    <core:text class="text-zinc-500">{{ __('agentic::agentic.tools.no_data') }}</core:text>
                </div>
            @endif
        </core:card>

        {{-- Server Breakdown (AC16) --}}
        <core:card class="p-6">
            <div class="flex items-center justify-between mb-4">
                <core:heading size="lg">{{ __('agentic::agentic.tools.server_breakdown') }}</core:heading>
            </div>
            @if($this->serverStats->count() > 0)
                <div class="space-y-4">
                    @foreach($this->serverStats as $serverStat)
                        @php
                            $maxCalls = $this->serverStats->max('total_calls');
                            $percentage = $maxCalls > 0 ? ($serverStat->total_calls / $maxCalls) * 100 : 0;
                        @endphp
                        <div class="space-y-2">
                            <div class="flex items-center justify-between">
                                <core:text class="font-medium truncate">{{ $serverStat->server_id }}</core:text>
                                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tools.calls', ['count' => number_format($serverStat->total_calls)]) }}</core:text>
                            </div>
                            <div class="w-full bg-zinc-200 dark:bg-zinc-700 rounded-full h-2">
                                <div class="bg-violet-500 h-2 rounded-full transition-all" style="width: {{ $percentage }}%"></div>
                            </div>
                            <div class="flex items-center justify-between text-xs">
                                <core:text size="xs" class="text-zinc-400">{{ __('agentic::agentic.tools.tools', ['count' => $serverStat->unique_tools]) }}</core:text>
                                <core:text size="xs" class="{{ $this->getSuccessRateColorClass($serverStat->success_rate) }}">{{ __('agentic::agentic.tools.success', ['rate' => $serverStat->success_rate]) }}</core:text>
                            </div>
                        </div>
                    @endforeach
                </div>
            @else
                <div class="flex flex-col items-center justify-center h-64">
                    <core:icon name="server" class="w-12 h-12 text-zinc-300 dark:text-zinc-600 mb-3" />
                    <core:text class="text-zinc-500">{{ __('agentic::agentic.tools.no_server_data') }}</core:text>
                </div>
            @endif
        </core:card>
    </div>

    {{-- Top Tools (AC14 + AC17) --}}
    <core:card class="mb-6">
        <div class="p-4 border-b border-zinc-200 dark:border-zinc-700 flex items-center justify-between">
            <core:heading size="lg">{{ __('agentic::agentic.tools.top_tools') }}</core:heading>
            <a href="{{ route('hub.agents.tools.calls') }}" wire:navigate>
                <core:button variant="ghost" size="sm">{{ __('agentic::agentic.actions.view_all_calls') }}</core:button>
            </a>
        </div>
        @if($this->topTools->count() > 0)
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.tool') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.server') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.calls') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.success_rate') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.tools.stats.errors') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.tools.avg_duration') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300"></th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->topTools as $tool)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                                <td class="p-4">
                                    <core:text class="font-medium">{{ $tool->tool_name }}</core:text>
                                </td>
                                <td class="p-4">
                                    <code class="text-xs bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">{{ $tool->server_id }}</code>
                                </td>
                                <td class="p-4 text-right">
                                    <core:text>{{ number_format($tool->total_calls) }}</core:text>
                                </td>
                                <td class="p-4 text-right">
                                    <core:text class="{{ $this->getSuccessRateColorClass($tool->success_rate) }}">{{ $tool->success_rate }}%</core:text>
                                </td>
                                <td class="p-4 text-right">
                                    @if($tool->total_errors > 0)
                                        <span class="text-red-500 font-medium">{{ number_format($tool->total_errors) }}</span>
                                    @else
                                        <core:text class="text-zinc-400">0</core:text>
                                    @endif
                                </td>
                                <td class="p-4 text-right">
                                    @if($tool->avg_duration)
                                        <core:text size="sm" class="text-zinc-500">{{ round($tool->avg_duration) < 1000 ? round($tool->avg_duration) . 'ms' : round($tool->avg_duration / 1000, 2) . 's' }}</core:text>
                                    @else
                                        <core:text size="sm" class="text-zinc-400">-</core:text>
                                    @endif
                                </td>
                                <td class="p-4 text-right">
                                    <a href="{{ route('hub.agents.tools.calls', ['tool' => $tool->tool_name, 'server' => $tool->server_id]) }}" wire:navigate>
                                        <core:button variant="ghost" size="sm" icon="arrow-right">{{ __('agentic::agentic.tools.drill_down') }}</core:button>
                                    </a>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        @else
            <div class="flex flex-col items-center py-12 text-center">
                <core:icon name="wrench" class="w-16 h-16 text-zinc-300 dark:text-zinc-600 mb-4" />
                <core:heading size="lg" class="text-zinc-600 dark:text-zinc-400">{{ __('agentic::agentic.tools.no_tool_usage') }}</core:heading>
                <core:text class="text-zinc-500 mt-2">{{ __('agentic::agentic.tools.tool_calls_appear') }}</core:text>
            </div>
        @endif
    </core:card>

    {{-- Recent Errors --}}
    @if($this->recentErrors->count() > 0)
        <core:card>
            <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                <core:heading size="lg">{{ __('agentic::agentic.tools.recent_errors') }}</core:heading>
            </div>
            <div class="divide-y divide-zinc-200 dark:divide-zinc-700">
                @foreach($this->recentErrors as $error)
                    <div class="p-4 hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                        <div class="flex items-start justify-between gap-4">
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-2 mb-1">
                                    <core:text class="font-medium">{{ $error->tool_name }}</core:text>
                                    <code class="text-xs bg-zinc-100 dark:bg-zinc-800 px-2 py-0.5 rounded">{{ $error->server_id }}</code>
                                </div>
                                <core:text size="sm" class="text-red-500 line-clamp-2">{{ $error->error_message ?? __('agentic::agentic.tools.unknown_error') }}</core:text>
                                @if($error->error_code)
                                    <core:text size="xs" class="text-zinc-400 mt-1">{{ __('agentic::agentic.tools.error_code', ['code' => $error->error_code]) }}</core:text>
                                @endif
                            </div>
                            <div class="text-right flex-shrink-0">
                                <core:text size="sm" class="text-zinc-500">{{ $error->created_at->diffForHumans() }}</core:text>
                                @if($error->workspace)
                                    <core:text size="xs" class="text-zinc-400">{{ $error->workspace->name }}</core:text>
                                @endif
                            </div>
                        </div>
                    </div>
                @endforeach
            </div>
        </core:card>
    @endif
</div>

@push('scripts')
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
document.addEventListener('alpine:init', () => {
    Alpine.data('chartComponent', (chartData) => ({
        chart: null,
        chartData: chartData,

        initChart() {
            const ctx = this.$refs.chart.getContext('2d');
            const isDark = document.documentElement.classList.contains('dark');

            this.chart = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: this.chartData.labels,
                    datasets: [
                        {
                            label: 'Total Calls',
                            data: this.chartData.calls,
                            borderColor: 'rgb(139, 92, 246)',
                            backgroundColor: 'rgba(139, 92, 246, 0.1)',
                            tension: 0.3,
                            fill: true,
                        },
                        {
                            label: 'Errors',
                            data: this.chartData.errors,
                            borderColor: 'rgb(239, 68, 68)',
                            backgroundColor: 'rgba(239, 68, 68, 0.1)',
                            tension: 0.3,
                            fill: false,
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    interaction: {
                        intersect: false,
                        mode: 'index',
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            grid: {
                                color: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)',
                            },
                            ticks: {
                                color: isDark ? 'rgba(255, 255, 255, 0.6)' : 'rgba(0, 0, 0, 0.6)',
                            }
                        },
                        x: {
                            grid: {
                                display: false,
                            },
                            ticks: {
                                color: isDark ? 'rgba(255, 255, 255, 0.6)' : 'rgba(0, 0, 0, 0.6)',
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            labels: {
                                color: isDark ? 'rgba(255, 255, 255, 0.8)' : 'rgba(0, 0, 0, 0.8)',
                            }
                        }
                    }
                }
            });
        }
    }));
});
</script>
@endpush
