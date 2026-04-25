<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Console\Command;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;

class BrainReindexCommand extends Command
{
    protected $signature = 'brain:reindex
        {--all : Reindex all memories instead of only unindexed ones}
        {--org= : Restrict reindexing to a single organisation scope}
        {--project= : Restrict reindexing to a single project scope}
        {--stale : Reindex stale memories where indexed_at is null or older than 14 days}
        {--dry-run : Print the number of matching memories without dispatching jobs}
        {--elastic-only : Refresh Elasticsearch documents only without regenerating embeddings}
        {--chunk=100 : Number of memories to process per chunk}';

    protected $description = 'Dispatch embedding jobs for OpenBrain memories that need indexing';

    public function handle(): int
    {
        $chunkSize = $this->chunkSize();

        if ($chunkSize === null) {
            return self::FAILURE;
        }

        $isReindexingAll = (bool) $this->option('all');
        $isStaleOnly = (bool) $this->option('stale');
        $isDryRun = (bool) $this->option('dry-run');
        $isElasticOnly = (bool) $this->option('elastic-only');
        $scope = $this->scopeLabel($isReindexingAll, $isStaleOnly);
        $query = $this->buildQuery($isReindexingAll, $isStaleOnly);
        $count = (clone $query)->count();

        if ($isDryRun) {
            $this->info("DRY RUN: {$count} brain memory record(s) match {$scope} reindex filters.");

            return self::SUCCESS;
        }

        $dispatched = 0;

        $query->chunkById($chunkSize, function (Collection $memories) use (&$dispatched, $isElasticOnly): void {
            foreach ($memories as $memory) {
                $this->dispatchReindex($memory, $isElasticOnly);
                $dispatched++;
            }
        });

        if ($dispatched === 0) {
            $this->info('No brain memories need reindexing.');

            return self::SUCCESS;
        }

        if ($isElasticOnly) {
            $this->info("Dispatched {$dispatched} brain memory elastic-only reindex job(s) for {$scope} memories.");

            return self::SUCCESS;
        }

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

    private function buildQuery(bool $isReindexingAll, bool $isStaleOnly): Builder
    {
        $query = BrainMemory::query();
        $org = $this->option('org');
        $project = $this->option('project');

        if (is_string($org) && $org !== '') {
            $query->where('org', $org);
        }

        if (is_string($project) && $project !== '') {
            $query->where('project', $project);
        }

        if ($isStaleOnly) {
            $query->where(function (Builder $builder): void {
                $builder->whereNull('indexed_at')
                    ->orWhere('indexed_at', '<', now()->subDays(14));
            });

            return $query;
        }

        if (! $isReindexingAll) {
            $query->whereNull('indexed_at');
        }

        return $query;
    }

    private function scopeLabel(bool $isReindexingAll, bool $isStaleOnly): string
    {
        if ($isStaleOnly) {
            return 'stale';
        }

        return $isReindexingAll ? 'all' : 'unindexed';
    }

    private function dispatchReindex(BrainMemory $memory, bool $isElasticOnly): void
    {
        if (! $isElasticOnly) {
            EmbedMemory::dispatch($memory->id);

            return;
        }

        $memoryId = $memory->id;

        dispatch(static function () use ($memoryId): void {
            $memory = BrainMemory::query()->find($memoryId);

            if (! $memory instanceof BrainMemory) {
                return;
            }

            app(BrainService::class)->elasticIndex($memory);

            if ($memory->indexed_at !== null) {
                $memory->update(['indexed_at' => now()]);
            }
        });
    }
}
