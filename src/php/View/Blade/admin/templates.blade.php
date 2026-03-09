<div class="p-6">
    {{-- Header --}}
    <div class="flex items-center justify-between mb-6">
        <div>
            <core:heading size="xl">{{ __('agentic::agentic.templates.title') }}</core:heading>
            <core:text class="text-zinc-500 mt-1">{{ __('agentic::agentic.templates.subtitle') }}</core:text>
        </div>
        <div class="flex items-center gap-2">
            <core:button wire:click="openImportModal" variant="ghost" icon="arrow-up-tray">
                {{ __('agentic::agentic.actions.import') }}
            </core:button>
            <core:button href="{{ route('hub.agents.plans') }}" variant="ghost" icon="arrow-left">
                {{ __('agentic::agentic.actions.back_to_plans') }}
            </core:button>
        </div>
    </div>

    {{-- Stats Cards --}}
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <core:card class="p-4">
            <core:text class="text-2xl font-semibold">{{ $this->stats['total'] }}</core:text>
            <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.templates.stats.templates') }}</core:text>
        </core:card>
        <core:card class="p-4">
            <core:text class="text-2xl font-semibold">{{ $this->stats['categories'] }}</core:text>
            <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.templates.stats.categories') }}</core:text>
        </core:card>
        <core:card class="p-4">
            <core:text class="text-2xl font-semibold">{{ $this->stats['total_phases'] }}</core:text>
            <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.templates.stats.total_phases') }}</core:text>
        </core:card>
        <core:card class="p-4">
            <core:text class="text-2xl font-semibold">{{ $this->stats['with_variables'] }}</core:text>
            <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.templates.stats.with_variables') }}</core:text>
        </core:card>
    </div>

    {{-- Filters --}}
    <core:card class="p-4 mb-6">
        <div class="flex flex-wrap items-center gap-4">
            <div class="flex-1 min-w-[200px]">
                <core:input wire:model.live.debounce.300ms="search" :placeholder="__('agentic::agentic.templates.search_placeholder')"
                            icon="magnifying-glass"/>
            </div>

            <core:select wire:model.live="category" class="w-48">
                <core:select.option value="">{{ __('agentic::agentic.filters.all_categories') }}</core:select.option>
                @foreach($this->categories as $cat)
                    <core:select.option value="{{ $cat }}">{{ ucfirst($cat) }}</core:select.option>
                @endforeach
            </core:select>

            @if($category || $search)
                <core:button wire:click="clearFilters" variant="ghost" size="sm">
                    {{ __('agentic::agentic.actions.clear_filters') }}
                </core:button>
            @endif
        </div>
    </core:card>

    {{-- Templates Grid --}}
    @if($this->templates->count() > 0)
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            @foreach($this->templates as $template)
                <core:card class="p-6 flex flex-col">
                    {{-- Header --}}
                    <div class="flex items-start justify-between mb-4">
                        <div class="flex-1">
                            <core:heading size="lg">{{ $template['name'] }}</core:heading>
                            <span class="inline-block px-2 py-0.5 text-xs font-medium rounded-full mt-2 {{ $this->getCategoryColor($template['category']) }}">
                                {{ ucfirst($template['category']) }}
                            </span>
                        </div>
                        <core:dropdown>
                            <core:button variant="ghost" size="sm" icon="ellipsis-vertical"/>
                            <core:menu>
                                <core:menu.item wire:click="openPreview('{{ $template['slug'] }}')" icon="eye">
                                    {{ __('agentic::agentic.actions.preview') }}
                                </core:menu.item>
                                <core:menu.item wire:click="openCreateModal('{{ $template['slug'] }}')" icon="plus">
                                    {{ __('agentic::agentic.actions.create_plan') }}
                                </core:menu.item>
                                <core:menu.separator/>
                                <core:menu.item
                                        wire:click="deleteTemplate('{{ $template['slug'] }}')"
                                        wire:confirm="{{ __('agentic::agentic.confirm.delete_template') }}"
                                        variant="danger"
                                        icon="trash"
                                >
                                    {{ __('agentic::agentic.actions.delete') }}
                                </core:menu.item>
                            </core:menu>
                        </core:dropdown>
                    </div>

                    {{-- Description --}}
                    @if($template['description'])
                        <core:text class="text-zinc-600 dark:text-zinc-400 mb-4 flex-1">
                            {{ $template['description'] }}
                        </core:text>
                    @else
                        <div class="flex-1"></div>
                    @endif

                    {{-- Meta --}}
                    <div class="flex items-center gap-4 text-sm text-zinc-500 mb-4">
                        <div class="flex items-center gap-1">
                            <core:icon name="queue-list" class="w-4 h-4"/>
                            <span>{{ __('agentic::agentic.templates.phases_count', ['count' => $template['phases_count']]) }}</span>
                        </div>
                        @if(count($template['variables']) > 0)
                            <div class="flex items-center gap-1">
                                <core:icon name="variable" class="w-4 h-4"/>
                                <span>{{ __('agentic::agentic.templates.variables_count', ['count' => count($template['variables'])]) }}</span>
                            </div>
                        @endif
                    </div>

                    {{-- Variables Preview --}}
                    @if(count($template['variables']) > 0)
                        <div class="border-t border-zinc-200 dark:border-zinc-700 pt-4 mb-4">
                            <core:text size="sm" class="font-medium mb-2">{{ __('agentic::agentic.templates.variables') }}:</core:text>
                            <div class="flex flex-wrap gap-2">
                                @foreach(array_slice($template['variables'], 0, 3) as $var)
                                    <span class="inline-flex items-center px-2 py-1 rounded bg-zinc-100 dark:bg-zinc-800 text-xs">
                                        <code>{{ $var['name'] }}</code>
                                        @if($var['required'])
                                            <span class="text-red-500 ml-1">*</span>
                                        @endif
                                    </span>
                                @endforeach
                                @if(count($template['variables']) > 3)
                                    <span class="text-xs text-zinc-500">{{ __('agentic::agentic.templates.more', ['count' => count($template['variables']) - 3]) }}</span>
                                @endif
                            </div>
                        </div>
                    @endif

                    {{-- Actions --}}
                    <div class="flex gap-2">
                        <core:button wire:click="openPreview('{{ $template['slug'] }}')" variant="ghost" class="flex-1">
                            {{ __('agentic::agentic.templates.preview') }}
                        </core:button>
                        <core:button wire:click="openCreateModal('{{ $template['slug'] }}')" variant="primary"
                                     class="flex-1">
                            {{ __('agentic::agentic.templates.use_template') }}
                        </core:button>
                    </div>
                </core:card>
            @endforeach
        </div>
    @else
        <core:card class="p-12">
            <div class="text-center">
                <core:icon name="document-text" class="w-12 h-12 text-zinc-400 mx-auto mb-4"/>
                <core:heading size="lg" class="mb-2">{{ __('agentic::agentic.templates.no_templates') }}</core:heading>
                <core:text class="text-zinc-500 mb-4">
                    @if($search || $category)
                        {{ __('agentic::agentic.templates.no_templates_filtered') }}
                    @else
                        {{ __('agentic::agentic.templates.no_templates_empty') }}
                    @endif
                </core:text>
                @if($search || $category)
                    <core:button wire:click="clearFilters" variant="ghost">
                        {{ __('agentic::agentic.actions.clear_filters') }}
                    </core:button>
                @else
                    <core:button wire:click="openImportModal" icon="arrow-up-tray">
                        {{ __('agentic::agentic.templates.import_template') }}
                    </core:button>
                @endif
            </div>
        </core:card>
    @endif

    {{-- Preview Modal --}}
    @if($showPreviewModal && $this->previewTemplate)
        <core:modal wire:model.self="showPreviewModal" class="max-w-4xl">
            <div class="p-6">
                <div class="flex items-start justify-between mb-6">
                    <div>
                        <core:heading size="xl">{{ $this->previewTemplate['name'] }}</core:heading>
                        <span class="inline-block px-2 py-0.5 text-xs font-medium rounded-full mt-2 {{ $this->getCategoryColor($this->previewTemplate['category']) }}">
                            {{ ucfirst($this->previewTemplate['category']) }}
                        </span>
                    </div>
                    <core:button wire:click="closePreview" variant="ghost" icon="x-mark"/>
                </div>

                @if($this->previewTemplate['description'])
                    <core:text class="text-zinc-600 dark:text-zinc-400 mb-6">
                        {{ $this->previewTemplate['description'] }}
                    </core:text>
                @endif

                {{-- Guidelines --}}
                @if(!empty($this->previewTemplate['guidelines']))
                    <div class="mb-6">
                        <core:heading size="sm" class="mb-2">{{ __('agentic::agentic.templates.guidelines') }}</core:heading>
                        <ul class="list-disc list-inside space-y-1 text-sm text-zinc-600 dark:text-zinc-400">
                            @foreach($this->previewTemplate['guidelines'] as $guideline)
                                <li>{{ $guideline }}</li>
                            @endforeach
                        </ul>
                    </div>
                @endif

                {{-- Phases --}}
                <div class="mb-6">
                    <core:heading size="sm" class="mb-4">{{ __('agentic::agentic.plan_detail.phases') }} ({{ count($this->previewTemplate['phases']) }})
                    </core:heading>
                    <div class="space-y-4">
                        @foreach($this->previewTemplate['phases'] as $index => $phase)
                            <div class="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4">
                                <div class="flex items-center gap-3 mb-2">
                                    <span class="flex items-center justify-center w-6 h-6 rounded-full bg-violet-100 dark:bg-violet-900/50 text-violet-600 dark:text-violet-400 text-xs font-medium">
                                        {{ $phase['order'] }}
                                    </span>
                                    <core:heading size="sm">{{ $phase['name'] }}</core:heading>
                                </div>
                                @if($phase['description'])
                                    <core:text size="sm"
                                               class="text-zinc-500 mb-3 ml-9">{{ $phase['description'] }}</core:text>
                                @endif
                                @if(!empty($phase['tasks']))
                                    <ul class="ml-9 space-y-1">
                                        @foreach($phase['tasks'] as $task)
                                            <li class="flex items-start gap-2 text-sm text-zinc-600 dark:text-zinc-400">
                                                <span class="text-zinc-400 mt-0.5">•</span>
                                                <span>{{ is_array($task) ? $task['name'] : $task }}</span>
                                            </li>
                                        @endforeach
                                    </ul>
                                @endif
                            </div>
                        @endforeach
                    </div>
                </div>

                {{-- Variables --}}
                @php
                    $template = app(\Core\Mod\Agentic\Services\PlanTemplateService::class)->get($previewSlug);
                    $variables = $template['variables'] ?? [];
                @endphp
                @if(!empty($variables))
                    <div class="mb-6">
                        <core:heading size="sm" class="mb-3">{{ __('agentic::agentic.templates.variables') }}</core:heading>
                        <div class="bg-zinc-50 dark:bg-zinc-800/50 rounded-lg p-4">
                            <table class="w-full text-sm">
                                <thead>
                                <tr class="text-left text-zinc-500 border-b border-zinc-200 dark:border-zinc-700">
                                    <th class="pb-2 font-medium">{{ __('agentic::agentic.templates.variable') }}</th>
                                    <th class="pb-2 font-medium">{{ __('agentic::agentic.plan_detail.description') }}</th>
                                    <th class="pb-2 font-medium">{{ __('agentic::agentic.templates.default') }}</th>
                                    <th class="pb-2 font-medium">{{ __('agentic::agentic.templates.required') }}</th>
                                </tr>
                                </thead>
                                <tbody>
                                @foreach($variables as $name => $config)
                                    <tr class="border-b border-zinc-100 dark:border-zinc-700/50 last:border-0">
                                        <td class="py-2 font-mono text-violet-600 dark:text-violet-400">{{ $name }}</td>
                                        <td class="py-2 text-zinc-600 dark:text-zinc-400">{{ $config['description'] ?? '-' }}</td>
                                        <td class="py-2 text-zinc-500">{{ $config['default'] ?? '-' }}</td>
                                        <td class="py-2">
                                            @if($config['required'] ?? false)
                                                <span class="text-red-500">{{ __('agentic::agentic.templates.yes') }}</span>
                                            @else
                                                <span class="text-zinc-400">{{ __('agentic::agentic.templates.no') }}</span>
                                            @endif
                                        </td>
                                    </tr>
                                @endforeach
                                </tbody>
                            </table>
                        </div>
                    </div>
                @endif

                <div class="flex justify-end gap-2">
                    <core:button wire:click="closePreview" variant="ghost">{{ __('agentic::agentic.actions.close') }}</core:button>
                    <core:button wire:click="openCreateModal('{{ $previewSlug }}')" variant="primary">
                        {{ __('agentic::agentic.templates.use_this_template') }}
                    </core:button>
                </div>
            </div>
        </core:modal>
    @endif

    {{-- Create Plan Modal --}}
    @if($showCreateModal && $this->createTemplate)
        <core:modal wire:model.self="showCreateModal" class="max-w-2xl">
            <div class="p-6">
                <core:heading size="xl" class="mb-2">{{ __('agentic::agentic.templates.create_from_template') }}</core:heading>
                <core:text class="text-zinc-500 mb-6">{{ __('agentic::agentic.templates.using_template', ['name' => $this->createTemplate['name']]) }}</core:text>

                <form wire:submit="createPlan" class="space-y-6">
                    {{-- Plan Title --}}
                    <div>
                        <core:input
                                wire:model="createTitle"
                                :label="__('agentic::agentic.templates.plan_title')"
                                :placeholder="__('agentic::agentic.templates.plan_title_placeholder')"
                        />
                        @error('createTitle')
                        <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text>
                        @enderror
                    </div>

                    {{-- Workspace --}}
                    <div>
                        <core:select wire:model="createWorkspaceId" :label="__('agentic::agentic.table.workspace')">
                            <core:select.option value="">Select workspace...</core:select.option>
                            @foreach($this->workspaces as $ws)
                                <core:select.option value="{{ $ws->id }}">{{ $ws->name }}</core:select.option>
                            @endforeach
                        </core:select>
                        @error('createWorkspaceId')
                        <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text>
                        @enderror
                    </div>

                    {{-- Variables --}}
                    @if(!empty($this->createTemplate['variables']))
                        <div>
                            <core:heading size="sm" class="mb-3">{{ __('agentic::agentic.templates.template_variables') }}</core:heading>
                            <div class="space-y-4">
                                @foreach($this->createTemplate['variables'] as $name => $config)
                                    <div>
                                        <core:input
                                                wire:model="createVariables.{{ $name }}"
                                                label="{{ ucfirst(str_replace('_', ' ', $name)) }}{{ ($config['required'] ?? false) ? ' *' : '' }}"
                                                placeholder="{{ $config['description'] ?? 'Enter value...' }}"
                                        />
                                        @if($config['description'] ?? null)
                                            <core:text size="xs"
                                                       class="text-zinc-500 mt-1">{{ $config['description'] }}</core:text>
                                        @endif
                                        @error("createVariables.{$name}")
                                        <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text>
                                        @enderror
                                    </div>
                                @endforeach
                            </div>
                        </div>
                    @endif

                    {{-- Activate Option --}}
                    <div class="flex items-center gap-2">
                        <core:checkbox wire:model="createActivate" id="createActivate"/>
                        <label for="createActivate" class="text-sm text-zinc-700 dark:text-zinc-300">
                            {{ __('agentic::agentic.templates.activate_immediately') }}
                        </label>
                    </div>

                    {{-- Preview --}}
                    @if($this->createPreview)
                        <div class="bg-zinc-50 dark:bg-zinc-800/50 rounded-lg p-4">
                            <core:heading size="sm" class="mb-2">{{ __('agentic::agentic.templates.preview') }}</core:heading>
                            <div class="text-sm text-zinc-600 dark:text-zinc-400">
                                <p class="mb-2"><strong>{{ __('agentic::agentic.plan_detail.phases') }}:</strong> {{ count($this->createPreview['phases']) }}</p>
                                <div class="flex flex-wrap gap-2">
                                    @foreach($this->createPreview['phases'] as $phase)
                                        <span class="px-2 py-1 bg-zinc-200 dark:bg-zinc-700 rounded text-xs">
                                            {{ $phase['name'] }}
                                        </span>
                                    @endforeach
                                </div>
                            </div>
                        </div>
                    @endif

                    @error('createVariables')
                    <core:text size="sm" class="text-red-500">{{ $message }}</core:text>
                    @enderror

                    <div class="flex justify-end gap-2 pt-4 border-t border-zinc-200 dark:border-zinc-700">
                        <core:button type="button" wire:click="closeCreateModal" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                        <core:button type="submit" variant="primary">{{ __('agentic::agentic.actions.create_plan') }}</core:button>
                    </div>
                </form>
            </div>
        </core:modal>
    @endif

    {{-- Import Modal --}}
    @if($showImportModal)
        <core:modal wire:model.self="showImportModal" class="max-w-xl">
            <div class="p-6">
                <core:heading size="xl" class="mb-2">{{ __('agentic::agentic.templates.import.title') }}</core:heading>
                <core:text class="text-zinc-500 mb-6">{{ __('agentic::agentic.templates.import.subtitle') }}</core:text>

                <form wire:submit="importTemplate" class="space-y-6">
                    {{-- File Upload --}}
                    <div>
                        <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                            {{ __('agentic::agentic.templates.import.file_label') }}
                        </label>
                        <div class="border-2 border-dashed border-zinc-300 dark:border-zinc-600 rounded-lg p-6 text-center">
                            <input
                                    type="file"
                                    wire:model="importFile"
                                    accept=".yaml,.yml"
                                    class="hidden"
                                    id="importFile"
                            />
                            <label for="importFile" class="cursor-pointer">
                                <core:icon name="document-arrow-up" class="w-10 h-10 text-zinc-400 mx-auto mb-2"/>
                                <core:text class="text-zinc-600 dark:text-zinc-400">
                                    {{ __('agentic::agentic.templates.import.file_prompt') }}
                                </core:text>
                                <core:text size="sm" class="text-zinc-400 mt-1">
                                    {{ __('agentic::agentic.templates.import.file_types') }}
                                </core:text>
                            </label>
                        </div>
                        <div wire:loading wire:target="importFile" class="mt-2">
                            <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.templates.import.processing') }}</core:text>
                        </div>
                    </div>

                    {{-- Error --}}
                    @if($importError)
                        <div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                            <core:text class="text-red-700 dark:text-red-300">{{ $importError }}</core:text>
                        </div>
                    @endif

                    {{-- Preview --}}
                    @if($importPreview)
                        <div class="bg-zinc-50 dark:bg-zinc-800/50 rounded-lg p-4">
                            <core:heading size="sm" class="mb-3">{{ __('agentic::agentic.templates.import.preview') }}</core:heading>
                            <dl class="grid grid-cols-2 gap-2 text-sm">
                                <dt class="text-zinc-500">{{ __('agentic::agentic.templates.import.name') }}</dt>
                                <dd class="font-medium">{{ $importPreview['name'] }}</dd>

                                <dt class="text-zinc-500">{{ __('agentic::agentic.templates.import.category') }}</dt>
                                <dd>
                                    <span class="inline-block px-2 py-0.5 text-xs font-medium rounded-full {{ $this->getCategoryColor($importPreview['category']) }}">
                                        {{ ucfirst($importPreview['category']) }}
                                    </span>
                                </dd>

                                <dt class="text-zinc-500">{{ __('agentic::agentic.templates.import.phases') }}</dt>
                                <dd>{{ $importPreview['phases_count'] }}</dd>

                                <dt class="text-zinc-500">{{ __('agentic::agentic.templates.import.variables') }}</dt>
                                <dd>{{ $importPreview['variables_count'] }}</dd>

                                @if($importPreview['description'])
                                    <dt class="text-zinc-500 col-span-2 mt-2">{{ __('agentic::agentic.templates.import.description') }}</dt>
                                    <dd class="col-span-2 text-zinc-600 dark:text-zinc-400">{{ $importPreview['description'] }}</dd>
                                @endif
                            </dl>
                        </div>

                        {{-- Filename --}}
                        <div>
                            <core:input
                                    wire:model="importFileName"
                                    :label="__('agentic::agentic.templates.import.filename_label')"
                                    :placeholder="__('agentic::agentic.templates.import.filename_placeholder')"
                            />
                            <core:text size="xs" class="text-zinc-500 mt-1">
                                {{ __('agentic::agentic.templates.import.will_be_saved', ['filename' => $importFileName]) }}
                            </core:text>
                            @error('importFileName')
                            <core:text size="sm" class="text-red-500 mt-1">{{ $message }}</core:text>
                            @enderror
                        </div>
                    @endif

                    <div class="flex justify-end gap-2 pt-4 border-t border-zinc-200 dark:border-zinc-700">
                        <core:button type="button" wire:click="closeImportModal" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                        <core:button type="submit" variant="primary" :disabled="!$importPreview">
                            {{ __('agentic::agentic.templates.import_template') }}
                        </core:button>
                    </div>
                </form>
            </div>
        </core:modal>
    @endif
</div>
