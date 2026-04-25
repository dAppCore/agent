<div class="space-y-6">
    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        <div class="flex flex-wrap items-end gap-4">
            <label class="block">
                <span class="text-sm font-medium text-zinc-700">Search</span>
                <input wire:model.live.debounce.250ms="search" type="search" class="mt-2 rounded-lg border border-zinc-300 px-3 py-2 text-sm">
            </label>

            <label class="block">
                <span class="text-sm font-medium text-zinc-700">Tier</span>
                <select wire:model.live="tierFilter" class="mt-2 rounded-lg border border-zinc-300 px-3 py-2 text-sm">
                    <option value="">All tiers</option>
                    <option value="free">Free</option>
                    <option value="apollo">Apollo</option>
                    <option value="hades">Hades</option>
                </select>
            </label>

            <label class="block">
                <span class="text-sm font-medium text-zinc-700">Verification</span>
                <select wire:model.live="verifiedFilter" class="mt-2 rounded-lg border border-zinc-300 px-3 py-2 text-sm">
                    <option value="">All users</option>
                    <option value="verified">Verified</option>
                    <option value="unverified">Unverified</option>
                </select>
            </label>
        </div>
    </div>

    @if($actionMessage !== '')
        <div class="rounded-lg border px-4 py-3 text-sm {{ $actionType === 'success' ? 'border-green-200 bg-green-50 text-green-900' : 'border-amber-200 bg-amber-50 text-amber-900' }}">
            {{ $actionMessage }}
        </div>
    @endif

    <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white">
        <table class="min-w-full divide-y divide-zinc-200 text-sm">
            <thead class="bg-zinc-50">
                <tr>
                    <th class="px-4 py-3 text-left"><button type="button" wire:click="sortBy('name')">Name</button></th>
                    <th class="px-4 py-3 text-left"><button type="button" wire:click="sortBy('email')">Email</button></th>
                    <th class="px-4 py-3 text-left">Tier</th>
                    <th class="px-4 py-3 text-left">Verified</th>
                    <th class="px-4 py-3 text-left"><button type="button" wire:click="sortBy('created_at')">Created</button></th>
                    <th class="px-4 py-3 text-left">Actions</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-zinc-100">
                @forelse($users as $user)
                    <tr>
                        <td class="px-4 py-3">
                            <a href="{{ \Core\Mod\Agentic\Website\Hub\Support\HubRouteNames::url('platform.user', ['id' => $user->id], '/hub/platform/user/'.$user->id) }}" class="font-medium text-violet-700">
                                {{ $user->name ?? 'Unknown User' }}
                            </a>
                        </td>
                        <td class="px-4 py-3 text-zinc-600">{{ $user->email }}</td>
                        <td class="px-4 py-3">{{ is_object($user->tier ?? null) ? $user->tier->value : ($user->tier ?? 'free') }}</td>
                        <td class="px-4 py-3">{{ $user->email_verified_at ? 'Yes' : 'No' }}</td>
                        <td class="px-4 py-3 text-zinc-600">{{ $user->created_at?->diffForHumans() ?? 'Unknown' }}</td>
                        <td class="px-4 py-3">
                            @if(!$user->email_verified_at)
                                <button type="button" class="text-violet-700" wire:click="verifyEmail({{ $user->id }})">Verify</button>
                            @endif
                        </td>
                    </tr>
                @empty
                    <tr>
                        <td colspan="6" class="px-4 py-8 text-center text-zinc-500">No users matched the current filters.</td>
                    </tr>
                @endforelse
            </tbody>
        </table>
    </div>

    <div>{{ $users->links() }}</div>
</div>
