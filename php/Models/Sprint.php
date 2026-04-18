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
 * Sprint — groups issues into time-boxed iterations.
 *
 * @property int $id
 * @property int|null $workspace_id
 * @property string $slug
 * @property string $title
 * @property string|null $description
 * @property string|null $goal
 * @property string $status
 * @property array|null $metadata
 * @property \Carbon\Carbon|null $started_at
 * @property \Carbon\Carbon|null $ended_at
 * @property \Carbon\Carbon|null $archived_at
 * @property \Carbon\Carbon|null $deleted_at
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class Sprint extends Model
{
    use BelongsToWorkspace;
    use LogsActivity;
    use SoftDeletes;

    protected $fillable = [
        'workspace_id',
        'slug',
        'title',
        'description',
        'goal',
        'status',
        'metadata',
        'started_at',
        'ended_at',
        'archived_at',
    ];

    protected $casts = [
        'metadata' => 'array',
        'started_at' => 'datetime',
        'ended_at' => 'datetime',
        'archived_at' => 'datetime',
    ];

    public const STATUS_PLANNING = 'planning';

    public const STATUS_ACTIVE = 'active';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_CANCELLED = 'cancelled';

    // Relationships

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function issues(): HasMany
    {
        return $this->hasMany(Issue::class);
    }

    // Scopes

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_ACTIVE);
    }

    public function scopePlanning(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_PLANNING);
    }

    public function scopeNotCancelled(Builder $query): Builder
    {
        return $query->where('status', '!=', self::STATUS_CANCELLED);
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
        $this->update([
            'status' => self::STATUS_ACTIVE,
            'started_at' => $this->started_at ?? now(),
        ]);

        return $this;
    }

    public function complete(): self
    {
        $this->update([
            'status' => self::STATUS_COMPLETED,
            'ended_at' => now(),
        ]);

        return $this;
    }

    public function cancel(?string $reason = null): self
    {
        $metadata = $this->metadata ?? [];
        if ($reason) {
            $metadata['cancel_reason'] = $reason;
        }

        $this->update([
            'status' => self::STATUS_CANCELLED,
            'ended_at' => now(),
            'archived_at' => now(),
            'metadata' => $metadata,
        ]);

        return $this;
    }

    public function getProgress(): array
    {
        $issues = $this->issues;
        $total = $issues->count();
        $closed = $issues->where('status', Issue::STATUS_CLOSED)->count();
        $inProgress = $issues->where('status', Issue::STATUS_IN_PROGRESS)->count();

        return [
            'total' => $total,
            'closed' => $closed,
            'in_progress' => $inProgress,
            'open' => $total - $closed - $inProgress,
            'percentage' => $total > 0 ? round(($closed / $total) * 100) : 0,
        ];
    }

    public function toMcpContext(): array
    {
        return [
            'slug' => $this->slug,
            'title' => $this->title,
            'description' => $this->description,
            'goal' => $this->goal,
            'status' => $this->status,
            'progress' => $this->getProgress(),
            'started_at' => $this->started_at?->toIso8601String(),
            'ended_at' => $this->ended_at?->toIso8601String(),
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }

    public function getActivitylogOptions(): LogOptions
    {
        return LogOptions::defaults()
            ->logOnly(['title', 'status'])
            ->logOnlyDirty()
            ->dontSubmitEmptyLogs();
    }
}
