<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Mod\Agentic\Database\Factories\AgentPlanFactory;
use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Support\Str;
use Spatie\Activitylog\LogOptions;
use Spatie\Activitylog\Traits\LogsActivity;

/**
 * Agent Plan - represents a structured work plan.
 *
 * Provides persistent task tracking across agent sessions,
 * enabling multi-agent handoff and context recovery.
 *
 * @property int $id
 * @property int $workspace_id
 * @property string $slug
 * @property string $title
 * @property string|null $description
 * @property string|null $context
 * @property array|null $phases
 * @property string $status
 * @property string|null $current_phase
 * @property array|null $metadata
 * @property string|null $source_file
 * @property \Carbon\Carbon|null $archived_at
 * @property \Carbon\Carbon|null $deleted_at
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class AgentPlan extends Model
{
    use BelongsToWorkspace;

    /** @use HasFactory<AgentPlanFactory> */
    use HasFactory;

    use LogsActivity;
    use SoftDeletes;

    protected static function newFactory(): AgentPlanFactory
    {
        return AgentPlanFactory::new();
    }

    protected $fillable = [
        'workspace_id',
        'slug',
        'title',
        'description',
        'context',
        'phases',
        'status',
        'current_phase',
        'metadata',
        'source_file',
        'archived_at',
        'template_version_id',
    ];

    protected $casts = [
        'context' => 'array',
        'phases' => 'array',
        'metadata' => 'array',
        'archived_at' => 'datetime',
    ];

    // Status constants
    public const STATUS_DRAFT = 'draft';

    public const STATUS_ACTIVE = 'active';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_ARCHIVED = 'archived';

    // Relationships
    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function agentPhases(): HasMany
    {
        return $this->hasMany(AgentPhase::class)->orderBy('order');
    }

    public function sessions(): HasMany
    {
        return $this->hasMany(AgentSession::class);
    }

    public function states(): HasMany
    {
        return $this->hasMany(WorkspaceState::class);
    }

    public function templateVersion(): BelongsTo
    {
        return $this->belongsTo(PlanTemplateVersion::class, 'template_version_id');
    }

    // Scopes
    public function scopeActive(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_ACTIVE);
    }

    public function scopeDraft(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_DRAFT);
    }

    public function scopeNotArchived(Builder $query): Builder
    {
        return $query->where('status', '!=', self::STATUS_ARCHIVED);
    }

    /**
     * Order by status using CASE statement with whitelisted values.
     *
     * This is a safe replacement for orderByRaw("FIELD(status, ...)") which
     * could be vulnerable to SQL injection if extended with user input.
     */
    public function scopeOrderByStatus(Builder $query, string $direction = 'asc'): Builder
    {
        return $query->orderByRaw('CASE status
            WHEN ? THEN 1
            WHEN ? THEN 2
            WHEN ? THEN 3
            WHEN ? THEN 4
            ELSE 5
        END '.($direction === 'desc' ? 'DESC' : 'ASC'), [self::STATUS_ACTIVE, self::STATUS_DRAFT, self::STATUS_COMPLETED, self::STATUS_ARCHIVED]);
    }

    // Helpers
    public static function generateSlug(string $title): string
    {
        $baseSlug = Str::slug($title);
        $slug = $baseSlug;
        $count = 1;

        while (static::where('slug', $slug)->exists()) {
            $slug = "{$baseSlug}-{$count}";
            $count++;
        }

        return $slug;
    }

    public function activate(): self
    {
        $this->update(['status' => self::STATUS_ACTIVE]);

        return $this;
    }

    public function complete(): self
    {
        $this->update(['status' => self::STATUS_COMPLETED]);

        return $this;
    }

    public function archive(?string $reason = null): self
    {
        $metadata = $this->metadata ?? [];
        if ($reason) {
            $metadata['archive_reason'] = $reason;
        }

        $this->update([
            'status' => self::STATUS_ARCHIVED,
            'archived_at' => now(),
            'metadata' => $metadata,
        ]);

        return $this;
    }

    public function setCurrentPhase(string|int $phase): self
    {
        $this->update(['current_phase' => (string) $phase]);

        return $this;
    }

    public function getCurrentPhase(): ?AgentPhase
    {
        if (! $this->current_phase) {
            return $this->agentPhases()->first();
        }

        return $this->agentPhases()
            ->where(function ($query) {
                $query->where('order', $this->current_phase)
                    ->orWhere('name', $this->current_phase);
            })
            ->first();
    }

    public function getProgress(): array
    {
        $phases = $this->agentPhases;
        $total = $phases->count();
        $completed = $phases->where('status', AgentPhase::STATUS_COMPLETED)->count();
        $inProgress = $phases->where('status', AgentPhase::STATUS_IN_PROGRESS)->count();

        return [
            'total' => $total,
            'completed' => $completed,
            'in_progress' => $inProgress,
            'pending' => $total - $completed - $inProgress,
            'percentage' => $total > 0 ? round(($completed / $total) * 100) : 0,
        ];
    }

    public function checkAllPhasesComplete(): bool
    {
        return $this->agentPhases()
            ->whereNotIn('status', [AgentPhase::STATUS_COMPLETED, AgentPhase::STATUS_SKIPPED])
            ->count() === 0;
    }

    public function getState(string $key): mixed
    {
        $state = $this->states()->where('key', $key)->first();

        return $state?->value;
    }

    public function setState(string $key, mixed $value, string $type = 'json', ?string $description = null): WorkspaceState
    {
        return $this->states()->updateOrCreate(
            ['key' => $key],
            [
                'value' => $value,
                'type' => $type,
                'description' => $description,
            ]
        );
    }

    public function toMarkdown(): string
    {
        $md = "# {$this->title}\n\n";

        if ($this->description) {
            $md .= "{$this->description}\n\n";
        }

        $progress = $this->getProgress();
        $md .= "**Status:** {$this->status} | **Progress:** {$progress['percentage']}% ({$progress['completed']}/{$progress['total']} phases)\n\n";

        if ($this->context) {
            $md .= "## Context\n\n{$this->context}\n\n";
        }

        $md .= "## Phases\n\n";

        foreach ($this->agentPhases as $phase) {
            $statusIcon = match ($phase->status) {
                AgentPhase::STATUS_COMPLETED => '✅',
                AgentPhase::STATUS_IN_PROGRESS => '🔄',
                AgentPhase::STATUS_BLOCKED => '🚫',
                AgentPhase::STATUS_SKIPPED => '⏭️',
                default => '⬜',
            };

            $md .= "### {$statusIcon} Phase {$phase->order}: {$phase->name}\n\n";

            if ($phase->description) {
                $md .= "{$phase->description}\n\n";
            }

            if ($phase->tasks) {
                foreach ($phase->tasks as $task) {
                    $taskStatus = ($task['status'] ?? 'pending') === 'completed' ? '✅' : '⬜';
                    $taskName = $task['name'] ?? $task;
                    $md .= "- {$taskStatus} {$taskName}\n";
                }
                $md .= "\n";
            }
        }

        return $md;
    }

    public function toMcpContext(): array
    {
        $progress = $this->getProgress();

        return [
            'slug' => $this->slug,
            'title' => $this->title,
            'description' => $this->description,
            'status' => $this->status,
            'current_phase' => $this->current_phase,
            'workspace_id' => $this->workspace_id,
            'progress' => $progress,
            'phases' => $this->agentPhases->map(fn ($p) => $p->toMcpContext())->all(),
            'metadata' => $this->metadata,
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }

    public function getActivitylogOptions(): LogOptions
    {
        return LogOptions::defaults()
            ->logOnly(['title', 'status', 'current_phase'])
            ->logOnlyDirty()
            ->dontSubmitEmptyLogs();
    }
}
