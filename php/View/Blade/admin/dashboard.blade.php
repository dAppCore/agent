<admin:module :title="__('agentic::agentic.dashboard.title')" :subtitle="__('agentic::agentic.dashboard.subtitle')">
    <x-slot:actions>
        <core:button wire:click="refresh" wire:loading.attr="disabled" variant="ghost" icon="arrow-path">
            {{ __('agentic::agentic.actions.refresh') }}
        </core:button>
    </x-slot:actions>

    <admin:stats :items="$this->statCards" />

    @if($this->blockedAlert)
        <admin:alert
            :type="$this->blockedAlert['type']"
            :title="$this->blockedAlert['title']"
            :message="$this->blockedAlert['message']"
            :action="$this->blockedAlert['action']"
        />
    @endif

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <admin:activity-feed
            :title="__('agentic::agentic.dashboard.recent_activity')"
            :items="$this->activityItems"
            :empty="__('agentic::agentic.dashboard.no_activity')"
            emptyIcon="clipboard-document-list"
        />

        <admin:progress-list
            :title="__('agentic::agentic.dashboard.top_tools')"
            :action="route('hub.agents.tools')"
            :items="$this->toolItems"
            :empty="__('agentic::agentic.dashboard.no_tool_usage')"
            emptyIcon="wrench"
        />
    </div>

    <admin:link-grid :items="$this->quickLinks" />
</admin:module>
