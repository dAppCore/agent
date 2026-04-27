<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Livewire\Livewire;

uses(\Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase::class);

beforeEach(function (): void {
    $this->actingAsHades();
});

it('wires brain actions and flux blade controls', function (): void {
    $this->assertFluxComponentWiring(
        'BrainExplorer',
        'brain-explorer',
        ['ForgetKnowledge', 'ListKnowledge', 'RecallKnowledge'],
        ['<flux:card', 'wire:submit="searchMemories"', 'wire:click="forgetMemory'],
    );
});

it('renders recent memories when no query is provided', function (): void {
    $component = $this->livewireComponent('BrainExplorer');
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
    $component = $this->livewireComponent('BrainExplorer');
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
