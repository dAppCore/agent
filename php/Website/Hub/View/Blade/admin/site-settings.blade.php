<div class="space-y-6">
    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        <div class="flex flex-wrap items-center gap-3">
            <div>
                <h2 class="text-lg font-semibold">{{ $this->workspace?->name ?? $workspaceSlug }}</h2>
                <p class="mt-1 text-sm text-zinc-600">Per-workspace configuration surface for tabs, deployment and connector settings.</p>
            </div>
        </div>

        <div class="mt-4 flex flex-wrap gap-2">
            @foreach($this->tabs as $tabKey => $tabData)
                <a
                    href="{{ $tabData['href'] }}"
                    class="rounded-full px-3 py-1.5 text-sm {{ $tab === $tabKey ? 'bg-violet-100 text-violet-700' : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200' }}"
                >
                    {{ $tabData['label'] }}
                </a>
            @endforeach
        </div>
    </div>

    <div class="rounded-2xl border border-zinc-200 bg-white p-6">
        @if($tab === 'services')
            <h3 class="text-base font-semibold">Services</h3>
            <p class="mt-2 text-sm text-zinc-600">Service-specific admin pages are deferred. This foundation page keeps the tab routing and workspace authorisation in place.</p>
        @elseif($tab === 'general')
            <h3 class="text-base font-semibold">General</h3>
            <div class="mt-3 space-y-2 text-sm text-zinc-600">
                <div><span class="font-medium text-zinc-900">Slug:</span> {{ $workspaceSlug }}</div>
                <div><span class="font-medium text-zinc-900">Domain:</span> {{ $this->workspace?->domain ?? 'Not configured' }}</div>
            </div>
        @elseif($tab === 'deployment')
            <h3 class="text-base font-semibold">Deployment</h3>
            <p class="mt-2 text-sm text-zinc-600">Deployment history is deferred. This tab exists so the routed workspace settings surface matches the RFC.</p>
        @elseif($tab === 'environment')
            <h3 class="text-base font-semibold">Environment</h3>
            <p class="mt-2 text-sm text-zinc-600">Environment editing is deferred. Wiring remains ready for future Livewire forms.</p>
        @elseif($tab === 'ssl')
            <h3 class="text-base font-semibold">SSL & Security</h3>
            <p class="mt-2 text-sm text-zinc-600">The WordPress connector block below covers the webhook-oriented integration surface from the RFC foundation slice.</p>
            @if($this->workspace)
                <div class="mt-6">
                    @livewire(\Core\Mod\Agentic\Website\Hub\View\Modal\Admin\WpConnectorSettings::class, ['workspace' => $this->workspace], key('wp-connector-'.$this->workspace->id))
                </div>
            @endif
        @elseif($tab === 'backups')
            <h3 class="text-base font-semibold">Backups</h3>
            <p class="mt-2 text-sm text-zinc-600">Backup management is deferred.</p>
        @else
            <h3 class="text-base font-semibold">Danger Zone</h3>
            <p class="mt-2 text-sm text-zinc-600">Destructive workspace operations are intentionally deferred in this partial delivery.</p>
        @endif
    </div>
</div>
