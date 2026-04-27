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

class DeleteFromIndex implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $tries = 3;

    /**
     * @var array<int, int>
     */
    public array $backoff = [10, 60, 300];

    public function __construct(
        public string $memoryId,
        public bool $forceDeleteRecord = false,
    ) {}

    public function handle(BrainService $brain): void
    {
        $memory = BrainMemory::withTrashed()->find($this->memoryId);
        if ($memory instanceof BrainMemory && $memory->deleted_at === null) {
            return;
        }

        $brain->qdrantDelete([$this->memoryId]);
        $brain->elasticDelete($this->memoryId);

        if ($this->forceDeleteRecord && $memory instanceof BrainMemory && $memory->deleted_at !== null) {
            $memory->forceDelete();
        }
    }
}
