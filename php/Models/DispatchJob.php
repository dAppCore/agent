<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class DispatchJob extends Model
{
    use BelongsToWorkspace;
    use HasUuids;

    public $incrementing = false;

    protected $keyType = 'string';

    protected $fillable = [
        'workspace_id',
        'created_by',
        'repo',
        'org',
        'task',
        'agent_type',
        'template',
        'branch',
        'priority',
        'labels',
        'status',
        'assigned_agent',
        'assigned_at',
        'started_at',
        'completed_at',
        'result',
        'findings',
        'changes',
        'report',
        'metadata',
    ];

    protected $casts = [
        'priority' => 'integer',
        'labels' => 'array',
        'assigned_at' => 'datetime',
        'started_at' => 'datetime',
        'completed_at' => 'datetime',
        'result' => 'array',
        'findings' => 'array',
        'changes' => 'array',
        'report' => 'array',
        'metadata' => 'array',
    ];

    public const STATUS_PENDING = 'pending';

    public const STATUS_ASSIGNED = 'assigned';

    public const STATUS_RUNNING = 'running';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_FAILED = 'failed';

    /**
     * Taken back before it finished.
     *
     * Distinct from failed on purpose: a failure says the work was attempted
     * and did not succeed, which is worth investigating. A cancellation says
     * nobody wanted it any more, which is not.
     */
    public const STATUS_CANCELLED = 'cancelled';

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function scopePending(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_PENDING);
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->whereIn('status', [self::STATUS_ASSIGNED, self::STATUS_RUNNING]);
    }

    /**
     * @return array<string, mixed>
     */
    public function toApiPayload(): array
    {
        return [
            'job_id' => $this->id,
            'created_by' => $this->created_by,
            'repo' => $this->repo,
            'org' => $this->org,
            'task' => $this->task,
            'agent_type' => $this->agent_type,
            'template' => $this->template,
            'branch' => $this->branch,
            'priority' => $this->priority,
            'labels' => $this->labels ?? [],
            'status' => $this->status,
            'assigned_agent' => $this->assigned_agent,
            'assigned_at' => $this->assigned_at?->toIso8601String(),
            'started_at' => $this->started_at?->toIso8601String(),
            'completed_at' => $this->completed_at?->toIso8601String(),
            'result' => $this->result,
            'findings' => $this->findings,
            'changes' => $this->changes,
            'report' => $this->report,
            'metadata' => $this->metadata,
            'created_at' => $this->created_at?->toIso8601String(),
        ];
    }
}
