<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Carbon\Carbon;
use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
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
 * Issues nest. An epic is the long-haul unit — months, several disciplines —
 * and is not itself workable; it is closed by the closing of its children.
 * A child is an ordinary issue that happens to have a parent, so subtasks
 * cost nothing to mint and nest as deep as the work actually divides:
 *
 *   $epic = CreateIssue::run(['title' => 'Retire go.work', 'type' => 'epic'], 1);
 *   SplitIssue::run($epic, [
 *       ['title' => 'Map the dependency layers', 'discipline' => 'planning'],
 *       ['title' => 'Tag L0 modules',            'discipline' => 'code'],
 *       ['title' => 'Rewrite the release guide', 'discipline' => 'content'],
 *   ]);
 *
 * @property int $id
 * @property int|null $workspace_id
 * @property int|null $sprint_id
 * @property int|null $parent_id
 * @property string $slug
 * @property string $title
 * @property string|null $description
 * @property string $type
 * @property string|null $discipline
 * @property string $status
 * @property string $priority
 * @property array|null $labels
 * @property string|null $assignee
 * @property string|null $repository
 * @property string|null $reporter
 * @property array|null $metadata
 * @property Carbon|null $closed_at
 * @property Carbon|null $archived_at
 * @property Carbon|null $deleted_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
class Issue extends Model
{
    use BelongsToWorkspace;
    use LogsActivity;
    use SoftDeletes;

    protected $fillable = [
        'workspace_id',
        'sprint_id',
        'parent_id',
        'slug',
        'title',
        'description',
        'type',
        'discipline',
        'status',
        'priority',
        'labels',
        'assignee',
        'repository',
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

    public const TYPE_EPIC = 'epic';

    // Priority constants
    public const PRIORITY_LOW = 'low';

    public const PRIORITY_NORMAL = 'normal';

    public const PRIORITY_HIGH = 'high';

    public const PRIORITY_URGENT = 'urgent';

    // Discipline constants
    //
    // Discipline is orthogonal to type: it says what KIND of effort closes the
    // issue, not what kind of thing the issue is. An epic gathers children
    // across several of these — that spread is what makes it an epic rather
    // than a large task.
    public const DISCIPLINE_PLANNING = 'planning';

    public const DISCIPLINE_CODE = 'code';

    public const DISCIPLINE_CONTENT = 'content';

    public const DISCIPLINE_ANALYSIS = 'analysis';

    public const DISCIPLINE_CREATIVE = 'creative';

    public const DISCIPLINE_REVIEW = 'review';

    public const DISCIPLINE_OPS = 'ops';

    /**
     * Every discipline an issue may carry.
     *
     * @return list<string>
     */
    public static function disciplines(): array
    {
        return [
            self::DISCIPLINE_PLANNING,
            self::DISCIPLINE_CODE,
            self::DISCIPLINE_CONTENT,
            self::DISCIPLINE_ANALYSIS,
            self::DISCIPLINE_CREATIVE,
            self::DISCIPLINE_REVIEW,
            self::DISCIPLINE_OPS,
        ];
    }

    // Relationships

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function sprint(): BelongsTo
    {
        return $this->belongsTo(Sprint::class);
    }

    /**
     * The issue this one was split out of, if any.
     */
    public function parent(): BelongsTo
    {
        return $this->belongsTo(self::class, 'parent_id');
    }

    /**
     * The issues split out of this one — one level down only.
     */
    public function children(): HasMany
    {
        return $this->hasMany(self::class, 'parent_id');
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

    public function scopeEpics(Builder $query): Builder
    {
        return $query->where('type', self::TYPE_EPIC);
    }

    /**
     * Issues with no parent — what a board shows before anything is expanded.
     */
    public function scopeTopLevel(Builder $query): Builder
    {
        return $query->whereNull('parent_id');
    }

    public function scopeChildrenOf(Builder $query, int $parentId): Builder
    {
        return $query->where('parent_id', $parentId);
    }

    public function scopeOfDiscipline(Builder $query, string $discipline): Builder
    {
        return $query->where('discipline', $discipline);
    }

    public function scopeForRepository(Builder $query, string $repository): Builder
    {
        return $query->where('repository', $repository);
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

    public function isEpic(): bool
    {
        return $this->type === self::TYPE_EPIC;
    }

    /**
     * A subtask is any issue with a parent — position, not type.
     */
    public function isSubtask(): bool
    {
        return $this->parent_id !== null;
    }

    /**
     * Children that are neither closed nor archived, oldest first.
     *
     * @return Collection<int, self>
     */
    public function openChildren()
    {
        return $this->children()->notClosed()->orderBy('created_at')->get();
    }

    /**
     * How far through its children this issue is.
     *
     * An issue with no children reports total 0 and percent 100 — it has
     * nothing outstanding, so nothing blocks its close.
     *
     * @return array{total: int, closed: int, open: int, percent: int}
     */
    public function getProgress(): array
    {
        $total = $this->children()->count();
        $closed = $this->children()->closed()->count();

        return [
            'total' => $total,
            'closed' => $closed,
            'open' => $total - $closed,
            'percent' => $total === 0 ? 100 : (int) floor($closed / $total * 100),
        ];
    }

    /**
     * Close the issue, refusing while any child is still open.
     *
     * This is the whole point of the hierarchy: an epic is finished when the
     * separate pieces of work under it are finished, so closing one with work
     * outstanding is a mistake rather than a shortcut. Pass $force to override
     * — abandoning an epic is legitimate, it just should not be the default.
     *
     * @throws \RuntimeException when children are still open and $force is false
     */
    public function close(bool $force = false): self
    {
        if (! $force) {
            $open = $this->openChildren();

            if ($open->isNotEmpty()) {
                throw new \RuntimeException(sprintf(
                    "Cannot close issue '%s': %d child issue(s) still open (%s). Close them first, or pass force.",
                    $this->slug,
                    $open->count(),
                    $open->pluck('slug')->implode(', ')
                ));
            }
        }

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
            'discipline' => $this->discipline,
            'status' => $this->status,
            'priority' => $this->priority,
            'labels' => $this->labels ?? [],
            'assignee' => $this->assignee,
            'repository' => $this->repository,
            'reporter' => $this->reporter,
            'sprint' => $this->sprint?->slug,
            'parent' => $this->parent?->slug,
            'children' => $this->children()->pluck('slug')->all(),
            'progress' => $this->getProgress(),
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
            ->logOnly(['title', 'status', 'priority', 'assignee', 'sprint_id', 'parent_id', 'discipline'])
            ->logOnlyDirty()
            ->dontSubmitEmptyLogs();
    }
}
