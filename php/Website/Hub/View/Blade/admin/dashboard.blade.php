@php
    $cards = [
        ['label' => 'Workspaces', 'body' => 'Switch between workspaces and open the per-site settings surface.', 'href' => \Core\Mod\Agentic\Website\Hub\Support\HubRouteNames::url('sites', fallback: '/hub/workspaces')],
        ['label' => 'Usage', 'body' => 'Review storage, API calls, boosts and AI-service usage.', 'href' => \Core\Mod\Agentic\Website\Hub\Support\HubRouteNames::url('account.usage', fallback: '/hub/account/usage')],
        ['label' => 'Platform', 'body' => 'Hades-only user operations: search, verify and inspect accounts.', 'href' => \Core\Mod\Agentic\Website\Hub\Support\HubRouteNames::url('platform', fallback: '/hub/platform')],
    ];
@endphp

<div class="space-y-6">
    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        <h2 class="text-lg font-semibold">Hub foundation</h2>
        <p class="mt-2 text-sm text-zinc-600">This slice wires the shared Hub shell, search surface, workspace switching and the first load-bearing Livewire pages.</p>
    </div>

    <div class="grid gap-4 md:grid-cols-3">
        @foreach($cards as $card)
            <a href="{{ $card['href'] }}" class="rounded-2xl border border-zinc-200 bg-white p-5 transition hover:border-violet-300 hover:bg-violet-50">
                <div class="text-base font-semibold">{{ $card['label'] }}</div>
                <div class="mt-2 text-sm text-zinc-600">{{ $card['body'] }}</div>
            </a>
        @endforeach
    </div>
</div>
