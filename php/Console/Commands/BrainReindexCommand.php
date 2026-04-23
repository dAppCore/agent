<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Console\Command;

class BrainReindexCommand extends Command
{
    protected $signature = 'brain:reindex {--all} {--chunk=100}';

    protected $description = 'Dispatch embedding jobs for OpenBrain memories that need indexing';

    public function handle(): int
    {
        $chunkSize = $this->chunkSize();

        if ($chunkSize === null) {
            return self::FAILURE;
        }

        $isReindexingAll = (bool) $this->option('all');
        $query = BrainMemory::query();

        if (! $isReindexingAll) {
            $query->whereNull('indexed_at');
        }

        $dispatched = 0;

        $query->chunkById($chunkSize, static function ($memories) use (&$dispatched): void {
            foreach ($memories as $memory) {
                EmbedMemory::dispatch($memory->id);
                $dispatched++;
            }
        });

        if ($dispatched === 0) {
            $this->info('No brain memories need reindexing.');

            return self::SUCCESS;
        }

        $scope = $isReindexingAll ? 'all' : 'unindexed';
        $this->info("Dispatched {$dispatched} brain memory embedding job(s) for {$scope} memories.");

        return self::SUCCESS;
    }

    private function chunkSize(): ?int
    {
        $option = $this->option('chunk');
        $chunkSize = filter_var($option, FILTER_VALIDATE_INT);

        if ($chunkSize === false || $chunkSize < 1) {
            $this->error('--chunk must be a positive integer.');

            return null;
        }

        return $chunkSize;
    }
}
