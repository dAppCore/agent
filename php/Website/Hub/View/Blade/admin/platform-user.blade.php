<div class="space-y-6">
    @if($actionMessage !== '')
        <div class="rounded-lg border px-4 py-3 text-sm {{ $actionType === 'success' ? 'border-green-200 bg-green-50 text-green-900' : 'border-amber-200 bg-amber-50 text-amber-900' }}">
            {{ $actionMessage }}
        </div>
    @endif

    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        <div class="flex flex-wrap items-center justify-between gap-4">
            <div>
                <h2 class="text-xl font-semibold">{{ $userRecord->name }}</h2>
                <p class="mt-1 text-sm text-zinc-600">{{ $userRecord->email }}</p>
            </div>

            <div class="flex flex-wrap gap-2">
                @foreach(['overview', 'workspaces', 'security'] as $tab)
                    <button
                        type="button"
                        class="rounded-full px-3 py-1.5 text-sm {{ $activeTab === $tab ? 'bg-violet-100 text-violet-700' : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200' }}"
                        wire:click="setTab('{{ $tab }}')"
                    >
                        {{ ucfirst($tab) }}
                    </button>
                @endforeach
            </div>
        </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-2">
        <div class="rounded-2xl border border-zinc-200 bg-white p-6">
            <h3 class="text-base font-semibold">Tier</h3>
            <div class="mt-4 space-y-4">
                <select wire:model.live="editingTier" class="w-full rounded-lg border border-zinc-300 px-3 py-2 text-sm">
                    <option value="free">Free</option>
                    <option value="apollo">Apollo</option>
                    <option value="hades">Hades</option>
                </select>
                <button type="button" class="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white" wire:click="saveTier">Save tier</button>
            </div>
        </div>

        <div class="rounded-2xl border border-zinc-200 bg-white p-6">
            <h3 class="text-base font-semibold">Verification</h3>
            <div class="mt-4 space-y-4">
                <label class="inline-flex items-center gap-2 text-sm text-zinc-700">
                    <input type="checkbox" wire:model.live="editingVerified">
                    Email verified
                </label>
                <button type="button" class="rounded-lg border border-zinc-300 px-4 py-2 text-sm" wire:click="saveVerification">Save verification</button>
            </div>
        </div>
    </div>

    @if($activeTab === 'workspaces')
        <div class="rounded-2xl border border-zinc-200 bg-white p-6">
            <h3 class="text-base font-semibold">Workspaces</h3>
            <div class="mt-3 text-sm text-zinc-600">
                @php
                    $workspaces = method_exists($userRecord, 'hostWorkspaces') ? $userRecord->hostWorkspaces : (method_exists($userRecord, 'workspaces') ? $userRecord->workspaces : collect());
                @endphp
                @forelse($workspaces as $workspace)
                    <div class="rounded-lg border border-zinc-200 px-4 py-3">
                        <div class="font-medium text-zinc-900">{{ $workspace->name ?? $workspace->slug }}</div>
                        <div class="text-zinc-500">{{ $workspace->slug ?? 'workspace' }}</div>
                    </div>
                @empty
                    <div>No workspace records available.</div>
                @endforelse
            </div>
        </div>
    @elseif($activeTab === 'security')
        <div class="rounded-2xl border border-zinc-200 bg-white p-6">
            <h3 class="text-base font-semibold">Security</h3>
            <div class="mt-3 text-sm text-zinc-600">
                <div>Created: {{ $userRecord->created_at?->toDayDateTimeString() ?? 'Unknown' }}</div>
                <div>Verified: {{ $userRecord->email_verified_at?->toDayDateTimeString() ?? 'Not verified' }}</div>
            </div>
        </div>
    @endif
</div>
