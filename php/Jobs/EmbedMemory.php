<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Jobs;

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

class EmbedMemory implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $tries = 3;

    /**
     * @var array<int, int>
     */
    public array $backoff = [10, 60, 300];

    public function __construct(
        public string $memoryId,
    ) {}

    public function handle(BrainService $brain): void
    {
        $memory = BrainMemory::find($this->memoryId);

        if (! $memory instanceof BrainMemory) {
            return;
        }

        $vector = $brain->embed($memory->content);

        $payload = $brain->buildQdrantPayload($memory->id, [
            'workspace_id' => $memory->workspace_id,
            'org' => $memory->getAttribute('org'),
            'project' => $memory->project,
            'agent_id' => $memory->agent_id,
            'type' => $memory->type,
            'tags' => $memory->tags ?? [],
            'confidence' => $memory->confidence,
            'source' => $memory->source ?? 'manual',
            'content' => $memory->content,
            'created_at' => $memory->created_at?->toIso8601String(),
        ]);
        $payload['vector'] = $vector;

        $brain->qdrantUpsert([$payload]);
        $brain->elasticIndex($memory);

        $memory->update(['indexed_at' => now()]);
    }
}
