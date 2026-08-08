<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Jobs;

use Core\Mod\Content\Models\ContentTask;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Log;

class BatchContentGeneration implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $timeout = 600;

    public function __construct(
        public string $priority = 'normal',
        public int $batchSize = 10,
    ) {
        $this->onQueue('ai-batch');
    }

    public function handle(): void
    {
        // Get pending and scheduled tasks ready for processing
        $tasks = ContentTask::query()
            ->where(function ($query) {
                $query->where('status', ContentTask::STATUS_PENDING)
                    ->orWhere(function ($q) {
                        $q->where('status', ContentTask::STATUS_SCHEDULED)
                            ->where('scheduled_for', '<=', now());
                    });
            })
            ->where('priority', $this->priority)
            ->orderBy('created_at')
            ->limit($this->batchSize)
            ->get();

        if ($tasks->isEmpty()) {
            Log::info("BatchContentGeneration: No {$this->priority} priority tasks to process");

            return;
        }

        Log::info("BatchContentGeneration: Processing {$tasks->count()} {$this->priority} priority tasks");

        foreach ($tasks as $task) {
            ProcessContentTask::dispatch($task);
        }
    }

    /**
     * Get the tags that should be assigned to the job.
     */
    public function tags(): array
    {
        return [
            'batch-generation',
            "priority:{$this->priority}",
        ];
    }
}
