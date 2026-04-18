<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\Task;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class TaskCommandTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    public function test_it_updates_task_details(): void
    {
        $task = Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Original title',
            'description' => 'Original description',
            'priority' => 'normal',
            'category' => 'task',
            'file_ref' => 'app/Original.php',
            'line_ref' => 12,
            'status' => 'pending',
        ]);

        $this->artisan('task', [
            'action' => 'update',
            '--workspace' => $this->workspace->id,
            '--id' => $task->id,
            '--title' => 'Updated title',
            '--desc' => 'Updated description',
            '--priority' => 'urgent',
            '--category' => 'docs',
            '--file' => 'app/Updated.php',
            '--line' => 88,
        ])
            ->expectsOutputToContain('Updated: Updated title')
            ->assertSuccessful();

        $this->assertDatabaseHas('tasks', [
            'id' => $task->id,
            'workspace_id' => $this->workspace->id,
            'title' => 'Updated title',
            'description' => 'Updated description',
            'priority' => 'urgent',
            'category' => 'docs',
            'file_ref' => 'app/Updated.php',
            'line_ref' => 88,
            'status' => 'pending',
        ]);
    }

    public function test_it_toggles_task_completion(): void
    {
        $task = Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Toggle me',
            'status' => 'pending',
        ]);

        $this->artisan('task', [
            'action' => 'toggle',
            '--workspace' => $this->workspace->id,
            '--id' => $task->id,
        ])
            ->expectsOutputToContain('Toggled: Toggle me pending → done')
            ->assertSuccessful();

        $this->assertSame('done', $task->fresh()->status);

        $this->artisan('task', [
            'action' => 'toggle',
            '--workspace' => $this->workspace->id,
            '--id' => $task->id,
        ])
            ->expectsOutputToContain('Toggled: Toggle me done → pending')
            ->assertSuccessful();

        $this->assertSame('pending', $task->fresh()->status);
    }

    public function test_it_rejects_update_without_any_changes(): void
    {
        $task = Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'No change',
            'status' => 'pending',
        ]);

        $this->artisan('task', [
            'action' => 'update',
            '--workspace' => $this->workspace->id,
            '--id' => $task->id,
        ])
            ->expectsOutputToContain('Provide at least one field to update')
            ->assertFailed();

        $this->assertSame('No change', $task->fresh()->title);
        $this->assertSame('pending', $task->fresh()->status);
    }
}
