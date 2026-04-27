<div class="rounded-2xl border border-zinc-200 bg-zinc-50 p-5">
    <div class="flex items-start justify-between gap-4">
        <div>
            <h4 class="text-base font-semibold">WordPress Connector</h4>
            <p class="mt-1 text-sm text-zinc-600">Webhook URL, secret and connection checks for the workspace WordPress integration.</p>
        </div>

        <label class="inline-flex items-center gap-2 text-sm text-zinc-700">
            <input type="checkbox" wire:model.live="enabled">
            Enabled
        </label>
    </div>

    <div class="mt-4 grid gap-4 md:grid-cols-2">
        <label class="block space-y-2 text-sm">
            <span class="font-medium text-zinc-900">WordPress URL</span>
            <input wire:model.live="wordpressUrl" type="url" class="w-full rounded-lg border border-zinc-300 px-3 py-2">
            @error('wordpressUrl')
                <span class="text-red-600">{{ $message }}</span>
            @enderror
        </label>

        <div class="space-y-2 text-sm">
            <div>
                <span class="font-medium text-zinc-900">Webhook URL</span>
                <div class="mt-1 rounded-lg border border-zinc-200 bg-white px-3 py-2 text-zinc-600">{{ $this->webhookUrl ?: 'Generated after save' }}</div>
            </div>

            <div>
                <span class="font-medium text-zinc-900">Webhook secret</span>
                <div class="mt-1 rounded-lg border border-zinc-200 bg-white px-3 py-2 font-mono text-xs text-zinc-600">{{ $this->webhookSecret ?: 'Generated after save' }}</div>
            </div>
        </div>
    </div>

    <div class="mt-4 flex flex-wrap gap-3">
        <button type="button" class="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white" wire:click="save">Save connector</button>
        <button type="button" class="rounded-lg border border-zinc-300 px-4 py-2 text-sm" wire:click="regenerateSecret">Regenerate secret</button>
        <button type="button" class="rounded-lg border border-zinc-300 px-4 py-2 text-sm" wire:click="testConnection" wire:loading.attr="disabled">Test connection</button>
    </div>

    <div class="mt-4 grid gap-3 md:grid-cols-3 text-sm text-zinc-600">
        <div><span class="font-medium text-zinc-900">Verified:</span> {{ $this->isVerified ? 'Yes' : 'No' }}</div>
        <div><span class="font-medium text-zinc-900">Last sync:</span> {{ $this->lastSync ?? 'Never' }}</div>
        <div><span class="font-medium text-zinc-900">Test result:</span> {{ $testResult ?? 'Not run' }}</div>
    </div>
</div>
