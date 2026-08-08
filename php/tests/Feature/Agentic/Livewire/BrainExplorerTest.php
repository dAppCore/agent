<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Agentic\Livewire;

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase;
use Illuminate\Support\Facades\Http;
use Livewire\Livewire;
use RuntimeException;

/**
 * A classic PHPUnit class, not a Pest file: Pest binds Tests\TestCase to the
 * whole Feature directory, and a file-level uses(LivewireTestCase::class)
 * inside that directory is a second binding for the same file, which Pest
 * rejects outright (TestCaseAlreadyInUse) regardless of the inheritance
 * between the two. Class-style tests are not touched by uses()->in(), which
 * is why the thirteen sibling files in Feature/Livewire have always run.
 */
class BrainExplorerTest extends LivewireTestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        // Forgetting a memory deletes its point from Qdrant. Unfaked, that is a
        // real request to localhost:6334 which retries six times before failing,
        // so the test asserted nothing about the component and just timed out
        // against whatever was or was not listening on the developer's machine.
        Http::fake();

        $this->actingAsHades();
    }

    public function test_wires_brain_actions_and_flux_blade_controls(): void
    {
        $this->assertFluxComponentWiring(
            'BrainExplorer',
            'brain-explorer',
            ['ForgetKnowledge', 'ListKnowledge', 'RecallKnowledge'],
            ['<flux:card', 'wire:submit="searchMemories"', 'wire:click="forgetMemory'],
        );
    }

    public function test_renders_recent_memories_when_no_query_is_provided(): void
    {
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
    }

    public function test_falls_back_to_database_search_and_forgets_memories(): void
    {
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

        $this->assertNotNull(BrainMemory::withTrashed()->find($memory->id)?->deleted_at);
    }
}
