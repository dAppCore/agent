<div class="space-y-6">
    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        <div class="flex items-center justify-between gap-4">
            <div>
                <h2 class="text-lg font-semibold">Workspaces</h2>
                <p class="mt-1 text-sm text-zinc-600">Entry point to the tenant and per-site settings flow.</p>
            </div>

            <input
                wire:model.live.debounce.200ms="search"
                type="search"
                placeholder="Search workspaces"
                class="w-full max-w-xs rounded-lg border border-zinc-300 px-3 py-2 text-sm"
            />
        </div>
    </div>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        @forelse($this->filteredWorkspaces as $workspace)
            <button
                type="button"
                class="rounded-2xl border border-zinc-200 bg-white p-5 text-left transition hover:border-violet-300 hover:bg-violet-50"
                wire:click="openWorkspace('{{ $workspace['slug'] }}')"
            >
                <div class="text-base font-semibold text-zinc-900">{{ $workspace['name'] }}</div>
                <div class="mt-1 text-sm text-zinc-600">{{ $workspace['description'] ?? 'Workspace settings and service configuration.' }}</div>
                <div class="mt-4 text-xs uppercase tracking-[0.2em] text-zinc-500">{{ $workspace['slug'] }}</div>
            </button>
        @empty
            <div class="rounded-2xl border border-dashed border-zinc-300 bg-white p-6 text-sm text-zinc-500">
                No workspaces matched the current search.
            </div>
        @endforelse
    </div>
</div>
