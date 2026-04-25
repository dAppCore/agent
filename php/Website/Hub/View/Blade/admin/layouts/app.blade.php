@php
    $hubUser = auth()->user();
    $hubWorkspace = is_object($hubUser) && method_exists($hubUser, 'defaultHostWorkspace')
        ? $hubUser->defaultHostWorkspace()
        : null;
    $menu = app(\Core\Mod\Agentic\Mod\Admin\Menu\AdminMenuRegistry::class)->items($hubUser, $hubWorkspace);
@endphp
<!DOCTYPE html>
<html lang="{{ str_replace('_', '-', app()->getLocale()) }}">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>{{ $title ?? 'Hub' }}</title>
    @livewireStyles
</head>
<body class="bg-zinc-50 text-zinc-900">
    <div class="min-h-screen lg:grid lg:grid-cols-[17rem_1fr]">
        <aside class="border-b border-zinc-200 bg-white p-6 lg:border-b-0 lg:border-r">
            <div class="mb-6 flex items-center justify-between gap-3">
                <div>
                    <div class="text-xs font-semibold uppercase tracking-[0.2em] text-zinc-500">Hub</div>
                    <div class="text-lg font-semibold">Admin Panel</div>
                </div>
            </div>

            <div class="mb-6">
                @if(auth()->check())
                    @livewire(\Core\Mod\Agentic\Website\Hub\View\Modal\Admin\WorkspaceSwitcher::class)
                @endif
            </div>

            <nav class="space-y-6">
                @foreach($menu as $group)
                    <div class="space-y-2">
                        @if(empty($group['meta']['standalone']))
                            <div class="text-xs font-semibold uppercase tracking-[0.2em] text-zinc-500">
                                {{ $group['meta']['label'] ?? 'Menu' }}
                            </div>
                        @endif

                        <div class="space-y-1">
                            @foreach($group['items'] as $item)
                                <a
                                    href="{{ $item['href'] ?? '#' }}"
                                    class="block rounded-lg px-3 py-2 text-sm {{ !empty($item['active']) ? 'bg-violet-50 text-violet-700' : 'text-zinc-700 hover:bg-zinc-100' }}"
                                >
                                    {{ $item['label'] }}
                                </a>
                            @endforeach
                        </div>
                    </div>
                @endforeach
            </nav>
        </aside>

        <div class="min-w-0">
            <header class="border-b border-zinc-200 bg-white px-6 py-4">
                <div class="flex items-center justify-between gap-4">
                    <div>
                        <h1 class="text-xl font-semibold">{{ $title ?? 'Hub' }}</h1>
                    </div>

                    <button
                        type="button"
                        class="rounded-lg border border-zinc-300 px-3 py-2 text-sm text-zinc-700 hover:bg-zinc-100"
                        onclick="window.Livewire && window.Livewire.dispatch('open-global-search')"
                    >
                        Search
                        <span class="ml-2 rounded bg-zinc-100 px-1.5 py-0.5 text-xs text-zinc-500">Cmd+K</span>
                    </button>
                </div>
            </header>

            <main class="p-6">
                @if(session()->has('warning'))
                    <div class="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
                        {{ session('warning') }}
                    </div>
                @endif

                {{ $slot }}
            </main>
        </div>
    </div>

    @if(auth()->check())
        @livewire(\Core\Mod\Agentic\Website\Hub\View\Modal\Admin\GlobalSearch::class)
    @endif

    @livewireScripts
    <script>
        document.addEventListener('keydown', function (event) {
            if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
                event.preventDefault();
                if (window.Livewire) {
                    window.Livewire.dispatch('open-global-search');
                }
            }
        });

        window.addEventListener('navigate-to-url', function (event) {
            if (window.Livewire && typeof window.Livewire.navigate === 'function') {
                window.Livewire.navigate(event.detail.url);
                return;
            }

            window.location.href = event.detail.url;
        });
    </script>
</body>
</html>
