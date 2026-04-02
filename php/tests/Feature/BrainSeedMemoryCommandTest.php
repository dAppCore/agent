<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Mockery;
use Tests\TestCase;

class BrainSeedMemoryCommandTest extends TestCase
{
    public function test_it_recursively_imports_markdown_files_from_a_directory(): void
    {
        $workspaceId = 42;
        $scanPath = $this->createSeedMemoryFixture();

        $brain = Mockery::mock(BrainService::class);
        $brain->shouldReceive('ensureCollection')->once();
        $brain->shouldReceive('remember')
            ->twice()
            ->andReturnUsing(static fn (): BrainMemory => new BrainMemory());

        $this->app->instance(BrainService::class, $brain);

        $this->artisan('brain:seed-memory', [
            '--workspace' => $workspaceId,
            '--agent' => 'virgil',
            '--path' => $scanPath,
        ])
            ->expectsOutputToContain('Found 2 markdown file(s) to process.')
            ->expectsOutputToContain('Imported 2 memories, skipped 0.')
            ->assertSuccessful();
    }

    private function createSeedMemoryFixture(): string
    {
        $scanPath = sys_get_temp_dir().'/brain-seed-'.bin2hex(random_bytes(6));
        $nestedPath = $scanPath.'/nested';

        mkdir($nestedPath, 0777, true);

        file_put_contents($scanPath.'/MEMORY.md', "# Memory\n\n## Architecture\nUse Core.Process() for command execution.\n\n## Decision\nPrefer named actions.");
        file_put_contents($nestedPath.'/notes.md', "# Notes\n\n## Convention\nUse UK English in user-facing output.");
        file_put_contents($nestedPath.'/ignore.txt', 'This file should not be imported.');

        return $scanPath;
    }
}
