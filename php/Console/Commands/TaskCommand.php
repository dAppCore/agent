<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\Task;
use Illuminate\Console\Command;

class TaskCommand extends Command
{
    protected $signature = 'task
        {action=list : Action: list, add, update, toggle, done, start, remove, show}
        {--id= : Task ID for update/toggle/done/start/remove/show}
        {--title= : Task title for add}
        {--desc= : Task description}
        {--priority=normal : Priority: low, normal, high, urgent}
        {--category= : Category: feature, bug, task, docs}
        {--file= : File reference}
        {--line= : Line number reference}
        {--status= : Filter by status: pending, in_progress, done, all}
        {--limit=20 : Limit results}
        {--workspace= : Workspace ID to scope queries (required for multi-tenant safety)}';

    protected $description = 'Manage development tasks';

    protected ?int $workspaceId = null;

    public function handle(): int
    {
        // Resolve workspace from option or authenticated user
        $this->workspaceId = $this->resolveWorkspaceId();

        if ($this->workspaceId === null) {
            $this->error('Workspace context required. Use --workspace=ID or ensure user is authenticated.');

            return 1;
        }

        $action = $this->argument('action');

        return match ($action) {
            'list', 'ls' => $this->listTasks(),
            'add', 'new' => $this->addTask(),
            'update', 'edit' => $this->updateTask(),
            'toggle', 'flip' => $this->toggleTask(),
            'done', 'complete' => $this->completeTask(),
            'start', 'wip' => $this->startTask(),
            'remove', 'rm', 'delete' => $this->removeTask(),
            'show' => $this->showTask(),
            default => $this->showHelp(),
        };
    }

    /**
     * Resolve workspace ID from option or authenticated user.
     */
    protected function resolveWorkspaceId(): ?int
    {
        // Explicit workspace option takes precedence
        if ($workspaceOption = $this->option('workspace')) {
            return (int) $workspaceOption;
        }

        // Fall back to authenticated user's default workspace
        $user = auth()->user();
        if ($user && method_exists($user, 'defaultHostWorkspace')) {
            $workspace = $user->defaultHostWorkspace();

            return $workspace?->id;
        }

        return null;
    }

    protected function listTasks(): int
    {
        $query = Task::forWorkspace($this->workspaceId);

        $status = $this->option('status');
        if ($status && $status !== 'all') {
            $query->where('status', $status);
        } elseif (! $status) {
            $query->active(); // Default: show active only
        }

        if ($category = $this->option('category')) {
            $query->where('category', $category);
        }

        $tasks = $query->orderByPriority()
            ->orderByStatus()
            ->limit($this->option('limit'))
            ->get();

        if ($tasks->isEmpty()) {
            $this->info('No tasks found.');

            return 0;
        }

        $this->newLine();

        foreach ($tasks as $task) {
            $line = sprintf(
                ' %s %s #%d %s',
                $task->status_badge,
                $task->priority_badge,
                $task->id,
                $task->title
            );

            if ($task->category) {
                $line .= " [{$task->category}]";
            }

            if ($task->file_ref) {
                $ref = basename($task->file_ref);
                if ($task->line_ref) {
                    $ref .= ":{$task->line_ref}";
                }
                $line .= " <comment>($ref)</comment>";
            }

            $this->line($line);
        }

        $this->newLine();
        $pending = Task::forWorkspace($this->workspaceId)->pending()->count();
        $inProgress = Task::forWorkspace($this->workspaceId)->inProgress()->count();
        $this->comment(" {$pending} pending, {$inProgress} in progress");

        return 0;
    }

    protected function addTask(): int
    {
        $title = $this->option('title');

        if (! $title) {
            $title = $this->ask('Task title');
        }

        if (! $title) {
            $this->error('Title is required');

            return 1;
        }

        $task = Task::create([
            'workspace_id' => $this->workspaceId,
            'title' => $title,
            'description' => $this->option('desc'),
            'priority' => $this->option('priority'),
            'category' => $this->option('category'),
            'file_ref' => $this->option('file'),
            'line_ref' => $this->option('line'),
            'status' => 'pending',
        ]);

        $this->info("Created task #{$task->id}: {$task->title}");

        return 0;
    }

    protected function updateTask(): int
    {
        $task = $this->findTask('update');

        if (! $task) {
            return 1;
        }

        $updates = [];

        $title = $this->option('title');
        if ($title !== null) {
            $title = trim((string) $title);
            if ($title === '') {
                $this->error('Title cannot be empty');

                return 1;
            }

            $updates['title'] = $title;
        }

        $description = $this->option('desc');
        if ($description !== null) {
            $updates['description'] = (string) $description;
        }

        $priority = $this->option('priority');
        if ($this->input->hasParameterOption('--priority')) {
            $allowed = ['low', 'normal', 'high', 'urgent'];
            if (! in_array($priority, $allowed, true)) {
                $this->error('Priority must be one of: low, normal, high, urgent');

                return 1;
            }

            $updates['priority'] = $priority;
        }

        $category = $this->option('category');
        if ($category !== null) {
            $updates['category'] = (string) $category;
        }

        $file = $this->option('file');
        if ($file !== null) {
            $updates['file_ref'] = (string) $file;
        }

        $line = $this->option('line');
        if ($line !== null) {
            if (! is_numeric($line)) {
                $this->error('Line must be a number');

                return 1;
            }

            $updates['line_ref'] = (int) $line;
        }

        if ($updates === []) {
            $this->error('Provide at least one field to update: --title, --desc, --priority, --category, --file, or --line');

            return 1;
        }

        $task->update($updates);

        $fresh = $task->fresh();
        $this->info("Updated: {$fresh->title}");

        return 0;
    }

    protected function toggleTask(): int
    {
        $task = $this->findTask('toggle');

        if (! $task) {
            return 1;
        }

        $currentStatus = $task->status;
        $newStatus = $currentStatus === 'done' ? 'pending' : 'done';
        $task->update(['status' => $newStatus]);

        $fresh = $task->fresh();
        $this->info("Toggled: {$fresh->title} {$currentStatus} → {$fresh->status}");

        return 0;
    }

    protected function completeTask(): int
    {
        $task = $this->findTask('complete');

        if (! $task) {
            return 1;
        }

        $task->update(['status' => 'done']);
        $this->info("Completed: {$task->title}");

        return 0;
    }

    protected function startTask(): int
    {
        $task = $this->findTask('start');

        if (! $task) {
            return 1;
        }

        $task->update(['status' => 'in_progress']);
        $this->info("Started: {$task->title}");

        return 0;
    }

    protected function removeTask(): int
    {
        $task = $this->findTask('remove');

        if (! $task) {
            return 1;
        }

        $title = $task->title;
        $task->delete();
        $this->info("Removed: {$title}");

        return 0;
    }

    protected function showTask(): int
    {
        $task = $this->findTask('show');

        if (! $task) {
            return 1;
        }

        $this->newLine();
        $this->line(" <info>#{$task->id}</info> {$task->title}");
        $this->line(" Status: {$task->status}");
        $this->line(" Priority: {$task->priority}");

        if ($task->category) {
            $this->line(" Category: {$task->category}");
        }

        if ($task->description) {
            $this->newLine();
            $this->line(" {$task->description}");
        }

        if ($task->file_ref) {
            $this->newLine();
            $ref = $task->file_ref;
            if ($task->line_ref) {
                $ref .= ":{$task->line_ref}";
            }
            $this->comment(" File: {$ref}");
        }

        $this->newLine();
        $this->comment(" Created: {$task->created_at->diffForHumans()}");

        return 0;
    }

    /**
     * Find a task by ID, scoped to the current workspace.
     */
    protected function findTask(string $action): ?Task
    {
        $id = $this->option('id');

        if (! $id) {
            $id = $this->ask("Task ID to {$action}");
        }

        if (! $id) {
            $this->error('Task ID is required');

            return null;
        }

        // Always scope by workspace to prevent data leakage
        $task = Task::forWorkspace($this->workspaceId)->where('id', $id)->first();

        if (! $task) {
            $this->error("Task #{$id} not found");

            return null;
        }

        return $task;
    }

    protected function showHelp(): int
    {
        $this->newLine();
        $this->line(' <info>Task Manager</info>');
        $this->newLine();
        $this->line(' <comment>Usage:</comment>');
        $this->line('   php artisan task list                    List active tasks');
        $this->line('   php artisan task add --title="Fix bug"   Add a task');
        $this->line('   php artisan task update --id=1 --desc="..."  Update task details');
        $this->line('   php artisan task toggle --id=1          Toggle task completion');
        $this->line('   php artisan task start --id=1            Start working on task');
        $this->line('   php artisan task done --id=1             Complete a task');
        $this->line('   php artisan task show --id=1             Show task details');
        $this->line('   php artisan task remove --id=1           Remove a task');
        $this->newLine();
        $this->line(' <comment>Options:</comment>');
        $this->line('   --workspace=ID   Workspace ID (required if not authenticated)');
        $this->line('   --priority=urgent|high|normal|low');
        $this->line('   --category=feature|bug|task|docs');
        $this->line('   --file=path/to/file.php --line=42');
        $this->line('   --status=pending|in_progress|done|all');
        $this->newLine();

        return 0;
    }
}
