<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Support\Str;
use Spatie\Activitylog\LogOptions;
use Spatie\Activitylog\Traits\LogsActivity;

/**
 * Issue — tracks bugs, features, tasks, and improvements.
 *
 * @property int $id
 * @property int|null $workspace_id
 * @property int|null $sprint_id
 * @property string $slug
 * @property string $title
 * @property string|null $description
 * @property string $type
 * @property string $status
 * @property string $priority
 * @property array|null $labels
 * @property string|null $assignee
 * @property string|null $reporter
 * @property array|null $metadata
 * @property \Carbon\Carbon|null $closed_at
 * @property \Carbon\Carbon|null $archived_at
 * @property \Carbon\Carbon|null $deleted_at
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class Issue extends Model
{
    use BelongsToWorkspace;
    use LogsActivity;
    use SoftDeletes;

    protected $fillable = [
        'workspace_id',
        'sprint_id',
        'slug',
        'title',
        'description',
        'type',
        'status',
        'priority',
        'labels',
        'assignee',
        'reporter',
        'metadata',
        'closed_at',
        'archived_at',
    ];

    protected $casts = [
        'labels' => 'array',
        'metadata' => 'array',
        'closed_at' => 'datetime',
        'archived_at' => 'datetime',
    ];

    // Status constants
    public const STATUS_OPEN = 'open';

    public const STATUS_IN_PROGRESS = 'in_progress';

    public const STATUS_REVIEW = 'review';

    public const STATUS_CLOSED = 'closed';

    // Type constants
    public const TYPE_BUG = 'bug';

    public const TYPE_FEATURE = 'feature';

    public const TYPE_TASK = 'task';

    public const TYPE_IMPROVEMENT = 'improvement';

    // Priority constants
    public const PRIORITY_LOW = 'low';

    public const PRIORITY_NORMAL = 'normal';

    public const PRIORITY_HIGH = 'high';

    public const PRIORITY_URGENT = 'urgent';

    // Relationships

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function sprint(): BelongsTo
    {
        return $this->belongsTo(Sprint::class);
    }

    public function comments(): HasMany
    {
        return $this->hasMany(IssueComment::class)->orderBy('created_at');
    }

    // Scopes

    public function scopeOpen(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_OPEN);
    }

    public function scopeInProgress(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_IN_PROGRESS);
    }

    public function scopeClosed(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_CLOSED);
    }

    public function scopeNotClosed(Builder $query): Builder
    {
        return $query->where('status', '!=', self::STATUS_CLOSED);
    }

    public function scopeOfType(Builder $query, string $type): Builder
    {
        return $query->where('type', $type);
    }

    public function scopeOfPriority(Builder $query, string $priority): Builder
    {
        return $query->where('priority', $priority);
    }

    public function scopeForSprint(Builder $query, int $sprintId): Builder
    {
        return $query->where('sprint_id', $sprintId);
    }

    public function scopeWithLabel(Builder $query, string $label): Builder
    {
        return $query->whereJsonContains('labels', $label);
    }

    /**
     * Order by priority using CASE statement with whitelisted values.
     */
    public function scopeOrderByPriority(Builder $query, string $direction = 'asc'): Builder
    {
        return $query->orderByRaw('CASE priority
            WHEN ? THEN 1
            WHEN ? THEN 2
            WHEN ? THEN 3
            WHEN ? THEN 4
            ELSE 5
        END '.($direction === 'desc' ? 'DESC' : 'ASC'), [self::PRIORITY_URGENT, self::PRIORITY_HIGH, self::PRIORITY_NORMAL, self::PRIORITY_LOW]);
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

    public function close(): self
    {
        $this->update([
            'status' => self::STATUS_CLOSED,
            'closed_at' => now(),
        ]);

        return $this;
    }

    public function reopen(): self
    {
        $this->update([
            'status' => self::STATUS_OPEN,
            'closed_at' => null,
        ]);

        return $this;
    }

    public function archive(?string $reason = null): self
    {
        $metadata = $this->metadata ?? [];
        if ($reason) {
            $metadata['archive_reason'] = $reason;
        }

        $this->update([
            'status' => self::STATUS_CLOSED,
            'closed_at' => $this->closed_at ?? now(),
            'archived_at' => now(),
            'metadata' => $metadata,
        ]);

        return $this;
    }

    public function addLabel(string $label): self
    {
        $labels = $this->labels ?? [];
        if (! in_array($label, $labels, true)) {
            $labels[] = $label;
            $this->update(['labels' => $labels]);
        }

        return $this;
    }

    public function removeLabel(string $label): self
    {
        $labels = $this->labels ?? [];
        $labels = array_values(array_filter($labels, fn (string $l) => $l !== $label));
        $this->update(['labels' => $labels]);

        return $this;
    }

    public function toMcpContext(): array
    {
        return [
            'slug' => $this->slug,
            'title' => $this->title,
            'description' => $this->description,
            'type' => $this->type,
            'status' => $this->status,
            'priority' => $this->priority,
            'labels' => $this->labels ?? [],
            'assignee' => $this->assignee,
            'reporter' => $this->reporter,
            'sprint' => $this->sprint?->slug,
            'comments_count' => $this->comments()->count(),
            'metadata' => $this->metadata,
            'closed_at' => $this->closed_at?->toIso8601String(),
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }

    public function getActivitylogOptions(): LogOptions
    {
        return LogOptions::defaults()
            ->logOnly(['title', 'status', 'priority', 'assignee', 'sprint_id'])
            ->logOnlyDirty()
            ->dontSubmitEmptyLogs();
    }
}
