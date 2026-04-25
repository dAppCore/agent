<div class="space-y-6">
    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        <h2 class="text-lg font-semibold">Usage</h2>
        <p class="mt-1 text-sm text-zinc-600">Storage, API calls, seats and AI-service usage.</p>

        <div class="mt-4 flex flex-wrap gap-2">
            @foreach($this->allowedTabs as $tabKey)
                <a
                    href="{{ \Core\Mod\Agentic\Website\Hub\Support\HubRouteNames::url('account.usage', ['tab' => $tabKey], '/hub/account/usage?tab='.$tabKey) }}"
                    class="rounded-full px-3 py-1.5 text-sm {{ $tab === $tabKey ? 'bg-violet-100 text-violet-700' : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200' }}"
                >
                    {{ ucfirst($tabKey) }}
                </a>
            @endforeach
        </div>
    </div>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-2xl border border-zinc-200 bg-white p-5">
            <div class="text-sm text-zinc-500">Storage used</div>
            <div class="mt-2 text-2xl font-semibold">{{ $this->stats['storage_used'] }}</div>
        </div>
        <div class="rounded-2xl border border-zinc-200 bg-white p-5">
            <div class="text-sm text-zinc-500">API calls</div>
            <div class="mt-2 text-2xl font-semibold">{{ number_format((int) $this->stats['api_calls']) }}</div>
        </div>
        <div class="rounded-2xl border border-zinc-200 bg-white p-5">
            <div class="text-sm text-zinc-500">Seats</div>
            <div class="mt-2 text-2xl font-semibold">{{ $this->stats['seats'] }}</div>
        </div>
        <div class="rounded-2xl border border-zinc-200 bg-white p-5">
            <div class="text-sm text-zinc-500">Boosts</div>
            <div class="mt-2 text-2xl font-semibold">{{ $this->stats['boosts'] }}</div>
        </div>
    </div>

    @if($tab === 'ai')
        <div class="rounded-2xl border border-zinc-200 bg-white p-6">
            <h3 class="text-base font-semibold">AI services</h3>
            <p class="mt-2 text-sm text-zinc-600">The routed AI-services view remains deferred, but the RFC redirect target is live and reserved here.</p>
        </div>
    @elseif($tab === 'boosts')
        <div class="rounded-2xl border border-zinc-200 bg-white p-6">
            <h3 class="text-base font-semibold">Boosts</h3>
            <p class="mt-2 text-sm text-zinc-600">Boost purchasing and history are deferred.</p>
        </div>
    @endif
</div>
