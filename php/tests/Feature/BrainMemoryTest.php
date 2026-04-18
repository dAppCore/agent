<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Str;
use Tests\TestCase;

class BrainMemoryTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();

        $this->workspace = Workspace::factory()->create();
    }

    public function test_BrainMemory_toMcpContext_Good_IncludesSupersessionDepthAndEmptyTags(): void
    {
        $root = BrainMemory::create([
            'workspace_id' => $this->workspace->id,
            'agent_id' => 'virgil',
            'type' => 'decision',
            'content' => 'Prefer named actions for all agent capabilities.',
            'confidence' => 0.9,
        ]);

        $child = BrainMemory::create([
            'workspace_id' => $this->workspace->id,
            'agent_id' => 'virgil',
            'type' => 'decision',
            'content' => 'Keep the action registry namespaced.',
            'tags' => ['actions', 'registry'],
            'confidence' => 0.95,
            'supersedes_id' => $root->id,
        ]);

        $context = $child->toMcpContext();

        $this->assertSame(1, $context['supersedes_count']);
        $this->assertSame(['actions', 'registry'], $context['tags']);
        $this->assertNull($context['deleted_at']);
    }

    public function test_BrainMemory_toMcpContext_Bad_HandlesMissingSupersededMemory(): void
    {
        $memory = new BrainMemory([
            'workspace_id' => $this->workspace->id,
            'agent_id' => 'virgil',
            'type' => 'research',
            'content' => 'Missing ancestors should not break MCP output.',
            'confidence' => 0.8,
        ]);

        $memory->supersedes_id = Str::uuid()->toString();

        $context = $memory->toMcpContext();

        $this->assertSame(0, $context['supersedes_count']);
        $this->assertSame([], $context['tags']);
    }

    public function test_BrainMemory_toMcpContext_Ugly_WalksSoftDeletedAncestors(): void
    {
        $root = BrainMemory::create([
            'workspace_id' => $this->workspace->id,
            'agent_id' => 'virgil',
            'type' => 'decision',
            'content' => 'Start with the root memory.',
            'confidence' => 0.9,
        ]);

        $middle = BrainMemory::create([
            'workspace_id' => $this->workspace->id,
            'agent_id' => 'virgil',
            'type' => 'decision',
            'content' => 'Replace the root memory.',
            'confidence' => 0.9,
            'supersedes_id' => $root->id,
        ]);

        $middle->delete();

        $leaf = BrainMemory::create([
            'workspace_id' => $this->workspace->id,
            'agent_id' => 'virgil',
            'type' => 'decision',
            'content' => 'Replace the deleted middle memory.',
            'confidence' => 0.9,
            'supersedes_id' => $middle->id,
        ]);

        $context = $leaf->toMcpContext();

        $this->assertSame(2, $context['supersedes_count']);
        $this->assertNull($context['deleted_at']);

        $deletedContext = BrainMemory::withTrashed()->findOrFail($middle->id)->toMcpContext();

        $this->assertNotNull($deletedContext['deleted_at']);
        $this->assertSame(1, $deletedContext['supersedes_count']);
    }
}
