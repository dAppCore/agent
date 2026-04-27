<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console;

use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Foundation\Console\Kernel as ConsoleKernel;

class Kernel extends ConsoleKernel
{
    protected function schedule(Schedule $schedule): void
    {
        parent::schedule($schedule);

        $schedule->command('agentic:sync-profiles')->hourly();
        $schedule->command('agentic:dispatch-queue --limit=3')->everyFiveMinutes();
    }
}
