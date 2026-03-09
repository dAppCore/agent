<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Mod\Agentic\Database\Factories\AgentPhaseFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Facades\DB;

/**
 * Agent Phase - individual phase within a plan.
 *
 * Tracks tasks, dependencies, and completion status.
 * Supports blocking and skipping for workflow control.
 *
 * @property int $id
 * @property int $agent_plan_id
 * @property int $order
 * @property string $name
 * @property string|null $description
 * @property array|null $tasks
 * @property array|null $dependencies
 * @property string $status
 * @property array|null $completion_criteria
 * @property \Carbon\Carbon|null $started_at
 * @property \Carbon\Carbon|null $completed_at
 * @property array|null $metadata
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class AgentPhase extends Model
{
    /** @use HasFactory<AgentPhaseFactory> */
    use HasFactory;

    protected static function newFactory(): AgentPhaseFactory
    {
        return AgentPhaseFactory::new();
    }

    protected $fillable = [
        'agent_plan_id',
        'order',
        'name',
        'description',
        'tasks',
        'dependencies',
        'status',
        'completion_criteria',
        'started_at',
        'completed_at',
        'metadata',
    ];

    protected $casts = [
        'tasks' => 'array',
        'dependencies' => 'array',
        'completion_criteria' => 'array',
        'metadata' => 'array',
        'started_at' => 'datetime',
        'completed_at' => 'datetime',
    ];

    // Status constants
    public const STATUS_PENDING = 'pending';

    public const STATUS_IN_PROGRESS = 'in_progress';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_BLOCKED = 'blocked';

    public const STATUS_SKIPPED = 'skipped';

    // Relationships
    public function plan(): BelongsTo
    {
        return $this->belongsTo(AgentPlan::class, 'agent_plan_id');
    }

    // Scopes
    public function scopePending(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_PENDING);
    }

    public function scopeInProgress(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_IN_PROGRESS);
    }

    public function scopeCompleted(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_COMPLETED);
    }

    public function scopeBlocked(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_BLOCKED);
    }

    // Status helpers
    public function isPending(): bool
    {
        return $this->status === self::STATUS_PENDING;
    }

    public function isInProgress(): bool
    {
        return $this->status === self::STATUS_IN_PROGRESS;
    }

    public function isCompleted(): bool
    {
        return $this->status === self::STATUS_COMPLETED;
    }

    public function isBlocked(): bool
    {
        return $this->status === self::STATUS_BLOCKED;
    }

    public function isSkipped(): bool
    {
        return $this->status === self::STATUS_SKIPPED;
    }

    // Actions
    public function start(): self
    {
        $this->update([
            'status' => self::STATUS_IN_PROGRESS,
            'started_at' => now(),
        ]);

        // Update plan's current phase
        $this->plan->setCurrentPhase($this->order);

        return $this;
    }

    public function complete(): self
    {
        DB::transaction(function () {
            $this->update([
                'status' => self::STATUS_COMPLETED,
                'completed_at' => now(),
            ]);

            // Check if all phases complete
            if ($this->plan->checkAllPhasesComplete()) {
                $this->plan->complete();
            }
        });

        return $this;
    }

    public function block(?string $reason = null): self
    {
        $metadata = $this->metadata ?? [];
        if ($reason) {
            $metadata['block_reason'] = $reason;
        }

        $this->update([
            'status' => self::STATUS_BLOCKED,
            'metadata' => $metadata,
        ]);

        return $this;
    }

    public function skip(?string $reason = null): self
    {
        $metadata = $this->metadata ?? [];
        if ($reason) {
            $metadata['skip_reason'] = $reason;
        }

        $this->update([
            'status' => self::STATUS_SKIPPED,
            'metadata' => $metadata,
        ]);

        return $this;
    }

    public function reset(): self
    {
        $this->update([
            'status' => self::STATUS_PENDING,
            'started_at' => null,
            'completed_at' => null,
        ]);

        return $this;
    }

    /**
     * Add a checkpoint note to the phase metadata.
     */
    public function addCheckpoint(string $note, array $context = []): self
    {
        $metadata = $this->metadata ?? [];
        $checkpoints = $metadata['checkpoints'] ?? [];

        $checkpoints[] = [
            'note' => $note,
            'context' => $context,
            'timestamp' => now()->toIso8601String(),
        ];

        $metadata['checkpoints'] = $checkpoints;
        $this->update(['metadata' => $metadata]);

        return $this;
    }

    /**
     * Get all checkpoints for this phase.
     */
    public function getCheckpoints(): array
    {
        return $this->metadata['checkpoints'] ?? [];
    }

    // Task management
    public function getTasks(): array
    {
        return $this->tasks ?? [];
    }

    public function addTask(string $name, ?string $notes = null): self
    {
        $tasks = $this->tasks ?? [];
        $tasks[] = [
            'name' => $name,
            'status' => 'pending',
            'notes' => $notes,
        ];
        $this->update(['tasks' => $tasks]);

        return $this;
    }

    public function completeTask(int|string $taskIdentifier): self
    {
        $tasks = $this->tasks ?? [];

        foreach ($tasks as $i => $task) {
            $taskName = is_string($task) ? $task : ($task['name'] ?? '');

            if ($i === $taskIdentifier || $taskName === $taskIdentifier) {
                if (is_string($tasks[$i])) {
                    $tasks[$i] = ['name' => $tasks[$i], 'status' => 'completed'];
                } else {
                    $tasks[$i]['status'] = 'completed';
                }
                break;
            }
        }

        $this->update(['tasks' => $tasks]);

        return $this;
    }

    public function getTaskProgress(): array
    {
        $tasks = $this->tasks ?? [];
        $total = count($tasks);
        $completed = 0;

        foreach ($tasks as $task) {
            $status = is_string($task) ? 'pending' : ($task['status'] ?? 'pending');
            if ($status === 'completed') {
                $completed++;
            }
        }

        return [
            'total' => $total,
            'completed' => $completed,
            'remaining' => $total - $completed,
            'percentage' => $total > 0 ? round(($completed / $total) * 100) : 0,
        ];
    }

    public function getRemainingTasks(): array
    {
        $tasks = $this->tasks ?? [];
        $remaining = [];

        foreach ($tasks as $task) {
            $status = is_string($task) ? 'pending' : ($task['status'] ?? 'pending');
            if ($status !== 'completed') {
                $remaining[] = is_string($task) ? $task : ($task['name'] ?? 'Unknown task');
            }
        }

        return $remaining;
    }

    public function allTasksComplete(): bool
    {
        $progress = $this->getTaskProgress();

        return $progress['total'] > 0 && $progress['remaining'] === 0;
    }

    // Dependency checking
    public function checkDependencies(): array
    {
        $dependencies = $this->dependencies ?? [];

        if (empty($dependencies)) {
            return [];
        }

        $blockers = [];

        $deps = AgentPhase::whereIn('id', $dependencies)->get();

        foreach ($deps as $dep) {
            if (! $dep->isCompleted() && ! $dep->isSkipped()) {
                $blockers[] = [
                    'phase_id' => $dep->id,
                    'phase_order' => $dep->order,
                    'phase_name' => $dep->name,
                    'status' => $dep->status,
                ];
            }
        }

        return $blockers;
    }

    public function canStart(): bool
    {
        return $this->isPending() && empty($this->checkDependencies());
    }

    // Output helpers
    public function getStatusIcon(): string
    {
        return match ($this->status) {
            self::STATUS_COMPLETED => '✅',
            self::STATUS_IN_PROGRESS => '🔄',
            self::STATUS_BLOCKED => '🚫',
            self::STATUS_SKIPPED => '⏭️',
            default => '⬜',
        };
    }

    public function toMcpContext(): array
    {
        $taskProgress = $this->getTaskProgress();

        return [
            'id' => $this->id,
            'order' => $this->order,
            'name' => $this->name,
            'description' => $this->description,
            'status' => $this->status,
            'tasks' => $this->tasks,
            'task_progress' => $taskProgress,
            'remaining_tasks' => $this->getRemainingTasks(),
            'dependencies' => $this->dependencies,
            'dependency_blockers' => $this->checkDependencies(),
            'can_start' => $this->canStart(),
            'started_at' => $this->started_at?->toIso8601String(),
            'completed_at' => $this->completed_at?->toIso8601String(),
            'metadata' => $this->metadata,
        ];
    }
}
