<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\AgenticGenerateCommand;
use Core\Mod\Agentic\Console\Commands\GenerateCommand;
use Illuminate\Contracts\Console\Kernel;

test('agent foundation console wiring registers the canonical commands', function (): void {
    $commands = app(Kernel::class)->all();

    expect($commands)->toHaveKey('agentic:generate')
        ->and($commands['agentic:generate'])->toBeInstanceOf(GenerateCommand::class)
        ->and(collect($commands)->contains(static fn (object $command): bool => $command instanceof AgenticGenerateCommand))
        ->toBeFalse()
        ->and(array_keys($commands))->toContain('brain:clean')
        ->toContain('brain:prune')
        ->toContain('brain:reindex');
});
