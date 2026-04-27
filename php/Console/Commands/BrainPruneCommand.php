<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Console\Command;
use Illuminate\Database\Eloquent\Collection;

class BrainPruneCommand extends Command
{
    protected $signature = 'brain:prune
        {--older-than=90 : Permanently delete memories soft-deleted more than this many days ago}
        {--chunk=100 : Number of memories to process per chunk}
        {--dry-run : Preview stale memories without dispatching jobs or deleting records}';

    protected $description = 'Permanently delete stale soft-deleted OpenBrain memories';

    public function handle(): int
    {
        $olderThan = $this->positiveIntegerOption('older-than');
        $chunkSize = $this->positiveIntegerOption('chunk');

        if ($olderThan === null || $chunkSize === null) {
            return self::FAILURE;
        }

        $cutoff = now()->subDays($olderThan);
        $query = BrainMemory::onlyTrashed()->where('deleted_at', '<', $cutoff);
        $count = (clone $query)->count();

        if ($count === 0) {
            $this->info('No stale soft-deleted brain memories found.');

            return self::SUCCESS;
        }

        if ((bool) $this->option('dry-run')) {
            $this->info("DRY RUN: {$count} stale soft-deleted brain memory record(s) would be permanently deleted.");

            return self::SUCCESS;
        }

        $pruned = 0;

        $query->chunkById($chunkSize, function (Collection $memories) use (&$pruned): void {
            foreach ($memories as $memory) {
                DeleteFromIndex::dispatch((string) $memory->id, true);
                $pruned++;
            }
        });

        $this->info("Pruned {$pruned} stale soft-deleted brain memory record(s).");

        return self::SUCCESS;
    }

    private function positiveIntegerOption(string $name): ?int
    {
        $option = $this->option($name);
        $value = filter_var($option, FILTER_VALIDATE_INT);

        if ($value === false || $value < 1) {
            $this->error("--{$name} must be a positive integer.");

            return null;
        }

        return $value;
    }
}
