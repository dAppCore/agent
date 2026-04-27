<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Console\Command;
use Illuminate\Database\Eloquent\Collection;

class BrainCleanCommand extends Command
{
    protected $signature = 'brain:clean
        {--chunk=100 : Number of soft-deleted memories to process per chunk}
        {--dry-run : Preview cleanup jobs without dispatching them}';

    protected $description = 'Dispatch index cleanup jobs for soft-deleted OpenBrain memories';

    public function handle(): int
    {
        $chunkSize = (int) $this->option('chunk');

        if ($chunkSize < 1) {
            $this->error('--chunk must be greater than zero.');

            return self::FAILURE;
        }

        $count = BrainMemory::onlyTrashed()->count();

        if ($count === 0) {
            $this->info('No soft-deleted brain memories found.');

            return self::SUCCESS;
        }

        if ((bool) $this->option('dry-run')) {
            $this->info("DRY RUN: {$count} soft-deleted brain memory record(s) would be removed from indexes.");

            return self::SUCCESS;
        }

        $dispatched = 0;

        BrainMemory::onlyTrashed()->chunkById($chunkSize, function (Collection $memories) use (&$dispatched): void {
            foreach ($memories as $memory) {
                DeleteFromIndex::dispatch($memory->id);
                $dispatched++;
            }
        });

        $this->info("Dispatched {$dispatched} index cleanup job(s) for soft-deleted brain memories.");

        return self::SUCCESS;
    }
}
