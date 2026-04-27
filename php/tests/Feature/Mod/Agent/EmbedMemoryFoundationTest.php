<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Http;

function foundationEmbedMemoryService(): BrainService
{
    return new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
    );
}

test('agent foundation embed memory skips rows that are already indexed', function (): void {
    $workspace = createWorkspace();

    $memory = BrainMemory::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'architecture',
        'content' => 'Indexed memories should not be embedded again.',
        'confidence' => 0.9,
        'indexed_at' => now()->subMinute(),
    ]);

    Http::fake();

    (new EmbedMemory($memory->id))->handle(foundationEmbedMemoryService());

    expect($memory->fresh()?->indexed_at)->not->toBeNull();
    Http::assertNothingSent();
});
