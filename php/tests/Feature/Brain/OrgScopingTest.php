<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Queue;

function orgScopingBrainService(): BrainService
{
    return new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
        elasticsearchUrl: 'https://elasticsearch.test',
    );
}

function orgScopingMemory(int $workspaceId, array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $workspaceId,
        'agent_id' => 'virgil',
        'type' => 'context',
        'content' => 'Organisation-scoped OpenBrain memory.',
        'confidence' => 0.85,
        'org' => 'core',
        'project' => 'agent',
    ], $attributes));
}

test('OrgScoping_remember_recall_Good_persists_org_and_recalls_with_matching_org', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $brain = orgScopingBrainService();

    $memory = $brain->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'Core remembers its own scoped knowledge.',
        'org' => 'core',
        'project' => 'agent',
        'confidence' => 0.92,
    ]);

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.125)]),
        'https://qdrant.test/collections/openbrain/points/search' => Http::response([
            'result' => [
                ['id' => $memory->id, 'score' => 0.91],
            ],
        ]),
    ]);

    $result = $brain->recall('core scoped knowledge', 5, ['org' => 'core'], $workspace->id);

    expect($memory->fresh()?->getAttribute('org'))->toBe('core')
        ->and($result['memories'])->toHaveCount(1)
        ->and($result['memories'][0]['id'])->toBe($memory->id)
        ->and($result['memories'][0]['org'])->toBe('core');

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/search'
        && $request->method() === 'POST'
        && $request['filter']['must'] === [
            ['key' => 'workspace_id', 'match' => ['value' => $workspace->id]],
            ['key' => 'org', 'match' => ['value' => 'core']],
        ]);
});

test('OrgScoping_remember_recall_Bad_does_not_recall_across_org_boundaries', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $brain = orgScopingBrainService();

    $memory = $brain->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'Core-only memory should not leak into another organisation scope.',
        'org' => 'core',
        'project' => 'agent',
        'confidence' => 0.88,
    ]);

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.125)]),
        'https://qdrant.test/collections/openbrain/points/search' => Http::response(['result' => []]),
    ]);

    $result = $brain->recall('core-only memory', 5, ['org' => 'other-org'], $workspace->id);

    expect($memory->fresh()?->getAttribute('org'))->toBe('core')
        ->and($result['memories'])->toBe([])
        ->and($result['scores'])->toBe([]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/search'
        && $request->method() === 'POST'
        && $request['filter']['must'] === [
            ['key' => 'workspace_id', 'match' => ['value' => $workspace->id]],
            ['key' => 'org', 'match' => ['value' => 'other-org']],
        ]);
});

test('OrgScoping_list_Ugly_filters_memories_by_org', function (): void {
    $workspace = createWorkspace();
    $otherWorkspace = createWorkspace();
    $coreMemory = orgScopingMemory($workspace->id, ['content' => 'Core memory', 'org' => 'core']);
    orgScopingMemory($workspace->id, ['content' => 'Other org memory', 'org' => 'other-org']);
    orgScopingMemory($otherWorkspace->id, ['content' => 'Other workspace memory', 'org' => 'core']);

    $result = (new BrainList)->handle([
        'org' => 'core',
        'limit' => 10,
    ], [
        'workspace_id' => $workspace->id,
    ]);

    expect($result['success'])->toBeTrue()
        ->and($result['count'])->toBe(1)
        ->and($result['memories'])->toHaveCount(1)
        ->and($result['memories'][0]['id'])->toBe($coreMemory->id)
        ->and($result['memories'][0]['org'])->toBe('core');
});
