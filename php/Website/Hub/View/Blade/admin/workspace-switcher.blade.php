<div x-data="{ open: @entangle('open') }" class="relative">
    <button
        type="button"
        class="flex w-full items-center justify-between rounded-lg border border-zinc-300 bg-white px-3 py-2 text-left text-sm"
        @click="open = !open"
    >
        <span>
            <span class="block font-medium text-zinc-900">{{ $current['name'] ?? 'Workspace' }}</span>
            <span class="block text-xs text-zinc-500">{{ $current['slug'] ?? '' }}</span>
        </span>
        <span class="text-zinc-400">▼</span>
    </button>

    <div
        x-cloak
        x-show="open"
        @click.outside="open = false"
        class="absolute z-20 mt-2 w-full rounded-lg border border-zinc-200 bg-white p-2 shadow-lg"
    >
        @forelse($workspaces as $slug => $workspace)
            <button
                type="button"
                class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm hover:bg-zinc-100"
                wire:click="switchWorkspace('{{ $slug }}')"
            >
                <span>
                    <span class="block font-medium text-zinc-900">{{ $workspace['name'] }}</span>
                    <span class="block text-xs text-zinc-500">{{ $workspace['description'] ?? $workspace['slug'] }}</span>
                </span>
                @if(($current['slug'] ?? null) === $slug)
                    <span class="text-violet-600">Current</span>
                @endif
            </button>
        @empty
            <div class="px-3 py-2 text-sm text-zinc-500">No workspaces available.</div>
        @endforelse
    </div>
</div>
