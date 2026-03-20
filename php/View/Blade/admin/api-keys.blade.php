<div>
    {{-- Header --}}
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
        <div>
            <div class="flex items-center gap-3 mb-2">
                <a href="{{ route('hub.agents.index') }}" wire:navigate class="text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300">
                    <core:icon name="arrow-left" class="w-5 h-5" />
                </a>
                <core:heading size="xl">{{ __('agentic::agentic.api_keys.title') }}</core:heading>
            </div>
            <core:subheading>{{ __('agentic::agentic.api_keys.subtitle') }}</core:subheading>
        </div>
        <div class="flex items-center gap-2">
            <core:button wire:click="exportUsageCsv" variant="ghost" icon="arrow-down-tray">
                {{ __('agentic::agentic.actions.export_csv') }}
            </core:button>
            <core:button wire:click="openCreateModal" icon="plus">
                {{ __('agentic::agentic.actions.create_key') }}
            </core:button>
        </div>
    </div>

    {{-- Stats --}}
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-violet-500/10 flex items-center justify-center">
                    <core:icon name="key" class="w-5 h-5 text-violet-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.api_keys.stats.total_keys') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ $this->stats['total'] }}</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                    <core:icon name="check-circle" class="w-5 h-5 text-green-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.api_keys.stats.active') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ $this->stats['active'] }}</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center">
                    <core:icon name="x-circle" class="w-5 h-5 text-red-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.api_keys.stats.revoked') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ $this->stats['revoked'] }}</core:text>
                </div>
            </div>
        </core:card>

        <core:card class="p-4">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
                    <core:icon name="arrow-path" class="w-5 h-5 text-blue-500" />
                </div>
                <div>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.api_keys.stats.total_calls') }}</core:text>
                    <core:text class="text-lg font-semibold">{{ number_format($this->stats['total_calls']) }}</core:text>
                </div>
            </div>
        </core:card>
    </div>

    {{-- Filters --}}
    <core:card class="p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-4">
            <div class="w-full md:w-48">
                <core:select wire:model.live="workspace">
                    <option value="">{{ __('agentic::agentic.filters.all_workspaces') }}</option>
                    @foreach($this->workspaces as $ws)
                        <option value="{{ $ws->id }}">{{ $ws->name }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-40">
                <core:select wire:model.live="status">
                    <option value="">{{ __('agentic::agentic.filters.all_status') }}</option>
                    <option value="active">{{ __('agentic::agentic.filters.active') }}</option>
                    <option value="revoked">{{ __('agentic::agentic.filters.revoked') }}</option>
                    <option value="expired">{{ __('agentic::agentic.filters.expired') }}</option>
                </core:select>
            </div>
            @if($workspace || $status)
                <core:button wire:click="clearFilters" variant="ghost" icon="x-mark">
                    {{ __('agentic::agentic.actions.clear') }}
                </core:button>
            @endif
        </div>
    </core:card>

    {{-- Keys Table --}}
    <core:card>
        @if($this->keys->count() > 0)
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.name') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.workspace') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.status') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.permissions') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.rate_limit') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">IP Restrictions</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.usage') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.last_used') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.created') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300"></th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->keys as $key)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-800/50 {{ $key->isRevoked() ? 'opacity-60' : '' }}">
                                <td class="p-4">
                                    <core:text class="font-medium">{{ $key->name }}</core:text>
                                    <core:text size="xs" class="text-zinc-400 mt-1 font-mono">{{ $key->getMaskedKey() }}</core:text>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm" class="text-zinc-500">{{ $key->workspace?->name ?? 'N/A' }}</core:text>
                                </td>
                                <td class="p-4">
                                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusBadgeClass($key) }}">
                                        {{ $key->getStatusLabel() }}
                                    </span>
                                    @if($key->expires_at && !$key->isRevoked())
                                        <core:text size="xs" class="text-zinc-400 mt-1">{{ $key->getExpiresForHumans() }}</core:text>
                                    @endif
                                </td>
                                <td class="p-4">
                                    <div class="flex flex-wrap gap-1">
                                        @foreach(array_slice($key->permissions ?? [], 0, 2) as $perm)
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
                                                {{ Str::after($perm, '.') }}
                                            </span>
                                        @endforeach
                                        @if(count($key->permissions ?? []) > 2)
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-zinc-100 dark:bg-zinc-800 text-zinc-500">
                                                +{{ count($key->permissions) - 2 }}
                                            </span>
                                        @endif
                                    </div>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm">{{ number_format($key->rate_limit) }}/min</core:text>
                                </td>
                                <td class="p-4">
                                    @if($key->ip_restriction_enabled)
                                        <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300">
                                            <core:icon name="shield-check" class="w-3 h-3" />
                                            {{ $key->getIpWhitelistCount() }} IPs
                                        </span>
                                        @if($key->last_used_ip)
                                            <core:text size="xs" class="text-zinc-400 mt-1">Last: {{ $key->last_used_ip }}</core:text>
                                        @endif
                                    @else
                                        <span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-zinc-100 dark:bg-zinc-800 text-zinc-500">
                                            Disabled
                                        </span>
                                    @endif
                                </td>
                                <td class="p-4">
                                    <core:text size="sm">{{ number_format($key->call_count) }} calls</core:text>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm" class="text-zinc-500">{{ $key->getLastUsedForHumans() }}</core:text>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm" class="text-zinc-500">{{ $key->created_at->diffForHumans() }}</core:text>
                                </td>
                                <td class="p-4 text-right">
                                    @if(!$key->isRevoked())
                                        <core:dropdown>
                                            <core:button variant="ghost" size="sm" icon="ellipsis-vertical" />
                                            <core:menu>
                                                <core:menu.item wire:click="openEditModal({{ $key->id }})" icon="pencil">
                                                    {{ __('agentic::agentic.actions.edit') }}
                                                </core:menu.item>
                                                <core:menu.item wire:click="revokeKey({{ $key->id }})" icon="x-circle" variant="danger" wire:confirm="{{ __('agentic::agentic.confirm.revoke_key') }}">
                                                    {{ __('agentic::agentic.actions.revoke') }}
                                                </core:menu.item>
                                            </core:menu>
                                        </core:dropdown>
                                    @endif
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>

            {{-- Pagination --}}
            <div class="p-4 border-t border-zinc-200 dark:border-zinc-700">
                {{ $this->keys->links() }}
            </div>
        @else
            <div class="flex flex-col items-center py-12 text-center">
                <core:icon name="key" class="w-16 h-16 text-zinc-300 dark:text-zinc-600 mb-4" />
                <core:heading size="lg" class="text-zinc-600 dark:text-zinc-400">{{ __('agentic::agentic.api_keys.no_keys') }}</core:heading>
                <core:text class="text-zinc-500 mt-2">
                    @if($workspace || $status)
                        {{ __('agentic::agentic.api_keys.no_keys_filtered') }}
                    @else
                        {{ __('agentic::agentic.api_keys.no_keys_empty') }}
                    @endif
                </core:text>
                @if(!$workspace && !$status)
                    <core:button wire:click="openCreateModal" icon="plus" class="mt-4">
                        {{ __('agentic::agentic.actions.create_key') }}
                    </core:button>
                @endif
            </div>
        @endif
    </core:card>

    {{-- Create Key Modal --}}
    <core:modal wire:model.self="showCreateModal" class="max-w-lg">
        <div class="p-6">
            <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.api_keys.create.title') }}</core:heading>

            <div class="space-y-4">
                <div>
                    <core:label>{{ __('agentic::agentic.api_keys.create.key_name') }}</core:label>
                    <core:input wire:model="newKeyName" :placeholder="__('agentic::agentic.api_keys.create.key_name_placeholder')" />
                    @error('newKeyName') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                </div>

                <div>
                    <core:label>{{ __('agentic::agentic.api_keys.create.workspace') }}</core:label>
                    <core:select wire:model="newKeyWorkspace">
                        @foreach($this->workspaces as $ws)
                            <option value="{{ $ws->id }}">{{ $ws->name }}</option>
                        @endforeach
                    </core:select>
                    @error('newKeyWorkspace') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                </div>

                <div>
                    <core:label>{{ __('agentic::agentic.api_keys.create.permissions') }}</core:label>
                    <div class="space-y-2 mt-2 p-3 bg-zinc-50 dark:bg-zinc-800 rounded-lg">
                        @foreach($this->availablePermissions as $perm => $description)
                            <label class="flex items-start gap-3 cursor-pointer">
                                <input type="checkbox" wire:model="newKeyPermissions" value="{{ $perm }}" class="mt-1 rounded border-zinc-300 dark:border-zinc-600 text-violet-500 focus:ring-violet-500">
                                <div>
                                    <core:text size="sm" class="font-medium">{{ $perm }}</core:text>
                                    <core:text size="xs" class="text-zinc-500">{{ $description }}</core:text>
                                </div>
                            </label>
                        @endforeach
                    </div>
                    @error('newKeyPermissions') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                </div>

                <div>
                    <core:label>{{ __('agentic::agentic.api_keys.create.rate_limit') }}</core:label>
                    <core:input type="number" wire:model="newKeyRateLimit" min="1" max="10000" />
                    @error('newKeyRateLimit') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                </div>

                <div>
                    <core:label>{{ __('agentic::agentic.api_keys.create.expiry') }}</core:label>
                    <core:select wire:model="newKeyExpiry">
                        <option value="">{{ __('agentic::agentic.api_keys.create.never_expires') }}</option>
                        <option value="30days">{{ __('agentic::agentic.api_keys.create.30_days') }}</option>
                        <option value="90days">{{ __('agentic::agentic.api_keys.create.90_days') }}</option>
                        <option value="1year">{{ __('agentic::agentic.api_keys.create.1_year') }}</option>
                    </core:select>
                </div>

                {{-- IP Restrictions --}}
                <div class="border-t border-zinc-200 dark:border-zinc-700 pt-4 mt-4">
                    <div class="flex items-center justify-between mb-3">
                        <div>
                            <core:label>IP Restrictions</core:label>
                            <core:text size="xs" class="text-zinc-500">Limit API access to specific IP addresses</core:text>
                        </div>
                        <label class="relative inline-flex items-center cursor-pointer">
                            <input type="checkbox" wire:model.live="newKeyIpRestrictionEnabled" class="sr-only peer">
                            <div class="w-11 h-6 bg-zinc-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-violet-300 dark:peer-focus:ring-violet-800 rounded-full peer dark:bg-zinc-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-zinc-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-zinc-600 peer-checked:bg-violet-600"></div>
                        </label>
                    </div>

                    @if($newKeyIpRestrictionEnabled)
                        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-3">
                            <div class="flex items-start gap-2">
                                <core:icon name="exclamation-triangle" class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" />
                                <core:text size="xs" class="text-amber-700 dark:text-amber-300">
                                    When enabled, only requests from whitelisted IPs will be accepted. Make sure to add your IPs before enabling.
                                </core:text>
                            </div>
                        </div>

                        <div>
                            <div class="flex items-center justify-between mb-1">
                                <core:label>Allowed IPs / CIDRs</core:label>
                                <core:text size="xs" class="text-zinc-400">Your IP: {{ $this->currentUserIp }}</core:text>
                            </div>
                            <textarea
                                wire:model="newKeyIpWhitelist"
                                rows="4"
                                class="w-full rounded-lg border border-zinc-300 dark:border-zinc-600 bg-white dark:bg-zinc-800 px-3 py-2 text-sm font-mono focus:border-violet-500 focus:ring-violet-500"
                                placeholder="192.168.1.1&#10;10.0.0.0/8&#10;2001:db8::/32"
                            ></textarea>
                            <core:text size="xs" class="text-zinc-500 mt-1">One IP or CIDR per line. Supports IPv4 and IPv6.</core:text>
                            @error('newKeyIpWhitelist') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                        </div>
                    @endif
                </div>
            </div>

            <div class="flex justify-end gap-2 mt-6">
                <core:button wire:click="closeCreateModal" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                <core:button wire:click="createKey">{{ __('agentic::agentic.actions.create_key') }}</core:button>
            </div>
        </div>
    </core:modal>

    {{-- Created Key Display Modal --}}
    <core:modal wire:model.self="showCreatedKeyModal" class="max-w-lg">
        <div class="p-6">
            <div class="flex items-center gap-3 mb-4">
                <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                    <core:icon name="check-circle" class="w-5 h-5 text-green-500" />
                </div>
                <core:heading size="lg">{{ __('agentic::agentic.api_keys.created.title') }}</core:heading>
            </div>

            <div class="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
                <div class="flex items-start gap-2">
                    <core:icon name="exclamation-triangle" class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
                    <div>
                        <core:text size="sm" class="font-medium text-amber-700 dark:text-amber-300">{{ __('agentic::agentic.api_keys.created.copy_now') }}</core:text>
                        <core:text size="sm" class="text-amber-600 dark:text-amber-400">{{ __('agentic::agentic.api_keys.created.copy_warning') }}</core:text>
                    </div>
                </div>
            </div>

            <div class="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
                <core:label class="mb-2">{{ __('agentic::agentic.api_keys.created.your_key') }}</core:label>
                <div class="flex items-center gap-2">
                    <code class="flex-1 text-sm font-mono bg-white dark:bg-zinc-900 px-3 py-2 rounded border border-zinc-200 dark:border-zinc-700 break-all">{{ $createdPlainKey }}</code>
                    <core:button variant="ghost" icon="clipboard" x-on:click="navigator.clipboard.writeText('{{ $createdPlainKey }}')">
                        {{ __('agentic::agentic.actions.copy') }}
                    </core:button>
                </div>
            </div>

            <div class="mt-4 text-sm text-zinc-500">
                <core:text size="sm">{{ __('agentic::agentic.api_keys.created.usage_hint') }}</core:text>
                <code class="block mt-1 text-xs bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">Authorization: Bearer {{ $createdPlainKey }}</code>
            </div>

            <div class="flex justify-end mt-6">
                <core:button wire:click="closeCreatedKeyModal">{{ __('agentic::agentic.actions.done') }}</core:button>
            </div>
        </div>
    </core:modal>

    {{-- Edit Key Modal --}}
    @if($showEditModal && $this->editingKey)
        <core:modal wire:model.self="showEditModal" class="max-w-lg">
            <div class="p-6">
                <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.api_keys.edit.title') }}</core:heading>

                <div class="mb-4 p-3 bg-zinc-50 dark:bg-zinc-800 rounded-lg">
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.api_keys.edit.key') }}</core:text>
                    <core:text class="font-medium">{{ $this->editingKey->name }}</core:text>
                    <core:text size="xs" class="text-zinc-400 font-mono">{{ $this->editingKey->getMaskedKey() }}</core:text>
                </div>

                <div class="space-y-4">
                    <div>
                        <core:label>{{ __('agentic::agentic.api_keys.create.permissions') }}</core:label>
                        <div class="space-y-2 mt-2 p-3 bg-zinc-50 dark:bg-zinc-800 rounded-lg">
                            @foreach($this->availablePermissions as $perm => $description)
                                <label class="flex items-start gap-3 cursor-pointer">
                                    <input type="checkbox" wire:model="editingPermissions" value="{{ $perm }}" class="mt-1 rounded border-zinc-300 dark:border-zinc-600 text-violet-500 focus:ring-violet-500">
                                    <div>
                                        <core:text size="sm" class="font-medium">{{ $perm }}</core:text>
                                        <core:text size="xs" class="text-zinc-500">{{ $description }}</core:text>
                                    </div>
                                </label>
                            @endforeach
                        </div>
                        @error('editingPermissions') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                    </div>

                    <div>
                        <core:label>{{ __('agentic::agentic.api_keys.create.rate_limit') }}</core:label>
                        <core:input type="number" wire:model="editingRateLimit" min="1" max="10000" />
                        @error('editingRateLimit') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                    </div>

                    {{-- IP Restrictions --}}
                    <div class="border-t border-zinc-200 dark:border-zinc-700 pt-4 mt-4">
                        <div class="flex items-center justify-between mb-3">
                            <div>
                                <core:label>IP Restrictions</core:label>
                                <core:text size="xs" class="text-zinc-500">Limit API access to specific IP addresses</core:text>
                            </div>
                            <label class="relative inline-flex items-center cursor-pointer">
                                <input type="checkbox" wire:model.live="editingIpRestrictionEnabled" class="sr-only peer">
                                <div class="w-11 h-6 bg-zinc-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-violet-300 dark:peer-focus:ring-violet-800 rounded-full peer dark:bg-zinc-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-zinc-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-zinc-600 peer-checked:bg-violet-600"></div>
                            </label>
                        </div>

                        @if($editingIpRestrictionEnabled)
                            <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-3">
                                <div class="flex items-start gap-2">
                                    <core:icon name="exclamation-triangle" class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" />
                                    <core:text size="xs" class="text-amber-700 dark:text-amber-300">
                                        When enabled, only requests from whitelisted IPs will be accepted. Make sure to add your IPs before enabling.
                                    </core:text>
                                </div>
                            </div>

                            <div>
                                <div class="flex items-center justify-between mb-1">
                                    <core:label>Allowed IPs / CIDRs</core:label>
                                    <core:text size="xs" class="text-zinc-400">Your IP: {{ $this->currentUserIp }}</core:text>
                                </div>
                                <textarea
                                    wire:model="editingIpWhitelist"
                                    rows="4"
                                    class="w-full rounded-lg border border-zinc-300 dark:border-zinc-600 bg-white dark:bg-zinc-800 px-3 py-2 text-sm font-mono focus:border-violet-500 focus:ring-violet-500"
                                    placeholder="192.168.1.1&#10;10.0.0.0/8&#10;2001:db8::/32"
                                ></textarea>
                                <core:text size="xs" class="text-zinc-500 mt-1">One IP or CIDR per line. Supports IPv4 and IPv6.</core:text>
                                @error('editingIpWhitelist') <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text> @enderror
                            </div>

                            @if($this->editingKey?->last_used_ip)
                                <div class="mt-3 p-2 bg-zinc-50 dark:bg-zinc-800 rounded">
                                    <core:text size="xs" class="text-zinc-500">
                                        Last used from: <span class="font-mono">{{ $this->editingKey->last_used_ip }}</span>
                                    </core:text>
                                </div>
                            @endif
                        @endif
                    </div>
                </div>

                <div class="flex justify-end gap-2 mt-6">
                    <core:button wire:click="closeEditModal" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                    <core:button wire:click="updateKey">{{ __('agentic::agentic.actions.save_changes') }}</core:button>
                </div>
            </div>
        </core:modal>
    @endif
</div>
