<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Blade;
use Livewire\Livewire;

uses(\Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase::class);

if (! function_exists('prepareAgenticLivewireHarness')) {
    function prepareAgenticLivewireHarness(): void
    {
        $base = sys_get_temp_dir().'/agentic-livewire-stubs';
        $componentPath = $base.'/components';
        $hubPath = $base.'/hub/admin/layouts';

        if (! is_dir($componentPath)) {
            mkdir($componentPath, 0777, true);
        }

        if (! is_dir($hubPath)) {
            mkdir($hubPath, 0777, true);
        }

        file_put_contents($hubPath.'/app.blade.php', '{{ $slot }}');

        $stubs = [
            'badge.blade.php' => '<span {{ $attributes }}>{{ $slot }}</span>',
            'button.blade.php' => '<button {{ $attributes }}>{{ $slot }}</button>',
            'card.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'heading.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'input.blade.php' => '<input {{ $attributes }} />',
            'select.blade.php' => '<select {{ $attributes }}>{{ $slot }}</select>',
            'text.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'textarea.blade.php' => '<textarea {{ $attributes }}>{{ $slot }}</textarea>',
        ];

        foreach ($stubs as $file => $contents) {
            file_put_contents($componentPath.'/'.$file, $contents);
        }

        Blade::anonymousComponentPath($componentPath, 'flux');
        app('view')->addNamespace('hub', $base.'/hub');
    }
}

if (! function_exists('loadAgenticLivewireComponent')) {
    function loadAgenticLivewireComponent(string $component): string
    {
        $phpRoot = dirname(__DIR__, 4);
        require_once $phpRoot."/Agentic/Livewire/{$component}.php";

        return "Core\\Mod\\Agentic\\Livewire\\{$component}";
    }
}

beforeEach(function (): void {
    prepareAgenticLivewireHarness();
    $this->actingAsHades();
});

it('wires brain actions and flux blade controls', function (): void {
    $phpRoot = dirname(__DIR__, 4);
    $componentSource = file_get_contents($phpRoot.'/Agentic/Livewire/BrainExplorer.php');
    $bladeSource = file_get_contents($phpRoot.'/resources/views/livewire/agentic/brain-explorer.blade.php');

    expect($componentSource)
        ->toContain('ForgetKnowledge')
        ->toContain('ListKnowledge')
        ->toContain('RecallKnowledge');

    expect($bladeSource)
        ->toContain('<flux:card')
        ->toContain('wire:submit="searchMemories"')
        ->toContain('wire:click="forgetMemory');
});

it('renders recent memories when no query is provided', function (): void {
    $component = loadAgenticLivewireComponent('BrainExplorer');
    $workspace = createWorkspace();

    BrainMemory::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'decision',
        'content' => 'Dispatch decisions are stored in the queue log.',
        'confidence' => 0.9,
        'tags' => ['dispatch', 'queue'],
    ]);

    Livewire::test($component, ['workspaceId' => $workspace->id])
        ->assertSee('Brain Explorer')
        ->assertSee('Dispatch decisions are stored in the queue log.')
        ->assertSee('virgil');
});

it('falls back to database search and forgets memories', function (): void {
    $component = loadAgenticLivewireComponent('BrainExplorer');
    $workspace = createWorkspace();

    $memory = BrainMemory::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'context',
        'content' => 'Dispatch queue memory for local fallback search.',
        'confidence' => 0.7,
        'tags' => ['dispatch'],
    ]);

    app()->instance(BrainService::class, new class extends BrainService
    {
        public function recall(
            string $query,
            int $topK,
            array $filter,
            int $workspaceId,
            array $keywords = [],
            array $boostKeywords = [],
        ): array {
            throw new RuntimeException('Brain backend offline');
        }
    });

    Livewire::test($component, ['workspaceId' => $workspace->id])
        ->set('query', 'dispatch queue')
        ->call('searchMemories')
        ->assertSee('Dispatch queue memory for local fallback search.')
        ->call('forgetMemory', $memory->id)
        ->assertDontSee('Dispatch queue memory for local fallback search.');

    expect(BrainMemory::withTrashed()->find($memory->id)?->deleted_at)->not->toBeNull();
});
