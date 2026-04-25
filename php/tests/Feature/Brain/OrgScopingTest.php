<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Auth\Access\AuthorizationException;
use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request as ClientRequest;
use Illuminate\Http\Request as HttpRequest;
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

function orgScopingBindRequestContext(object $workspace, array $context = []): void
{
    $request = HttpRequest::create('/api/v1/mcp/tools/call', 'POST');
    $request->attributes->set('workspace', $workspace);
    $request->attributes->set('workspace_id', $workspace->id);
    $request->attributes->set('mcp_workspace_context', array_merge([
        'workspace_id' => $workspace->id,
        'workspace' => $workspace,
    ], $context));

    app()->instance('request', $request);
}

test('OrgScoping_remember_recall_Good_persists_org_and_recalls_with_matching_org', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $workspace->setAttribute('slug', 'core');
    orgScopingBindRequestContext($workspace);
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

    Http::assertSent(fn (ClientRequest $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/search'
        && $request->method() === 'POST'
        && $request['filter']['must'] === [
            ['key' => 'workspace_id', 'match' => ['value' => $workspace->id]],
            ['key' => 'org', 'match' => ['value' => 'core']],
        ]);
});

test('OrgScoping_recall_Bad_rejects_an_unauthorised_org_before_any_qdrant_query', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $workspace->setAttribute('slug', 'core');
    orgScopingBindRequestContext($workspace);
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

    expect($memory->fresh()?->getAttribute('org'))->toBe('core');

    expect(fn () => $brain->recall('core-only memory', 5, ['org' => 'other-org'], $workspace->id))
        ->toThrow(AuthorizationException::class, "Organisation scope 'other-org' is not authorised for this authenticated workspace.");

    Http::assertNothingSent();
});

test('OrgScoping_remember_Bad_rejects_an_unauthorised_org_without_inserting_a_memory', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $workspace->setAttribute('slug', 'core');
    orgScopingBindRequestContext($workspace);
    $brain = orgScopingBrainService();

    expect(fn () => $brain->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'This should never be stored for another organisation.',
        'org' => 'evil',
        'project' => 'agent',
        'confidence' => 0.9,
    ]))->toThrow(AuthorizationException::class, "Organisation scope 'evil' is not authorised for this authenticated workspace.");

    expect(BrainMemory::query()->count())->toBe(0);
    Queue::assertNotPushed(EmbedMemory::class);
});

test('OrgScoping_forget_Bad_rejects_forgetting_a_memory_from_another_org', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $workspace->setAttribute('slug', 'core');
    orgScopingBindRequestContext($workspace);
    $brain = orgScopingBrainService();
    $memory = orgScopingMemory($workspace->id, [
        'content' => 'Other org memory must not be forgotten by core.',
        'org' => 'evil',
    ]);

    expect(fn () => $brain->forget($memory->id))
        ->toThrow(AuthorizationException::class, "Organisation scope 'evil' is not authorised for this authenticated workspace.");

    expect(BrainMemory::query()->find($memory->id))->not->toBeNull();
    Queue::assertNotPushed(DeleteFromIndex::class);
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
