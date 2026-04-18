<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class FleetTask extends Model
{
    use BelongsToWorkspace;

    public const STATUS_QUEUED = 'queued';

    public const STATUS_ASSIGNED = 'assigned';

    public const STATUS_IN_PROGRESS = 'in_progress';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_FAILED = 'failed';

    protected $fillable = [
        'workspace_id',
        'fleet_node_id',
        'repo',
        'branch',
        'task',
        'template',
        'agent_model',
        'status',
        'result',
        'findings',
        'changes',
        'report',
        'started_at',
        'completed_at',
    ];

    protected $casts = [
        'result' => 'array',
        'findings' => 'array',
        'changes' => 'array',
        'report' => 'array',
        'started_at' => 'datetime',
        'completed_at' => 'datetime',
    ];

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function fleetNode(): BelongsTo
    {
        return $this->belongsTo(FleetNode::class);
    }

    public function scopePendingForNode(Builder $query, FleetNode $node): Builder
    {
        return $query->where('fleet_node_id', $node->id)
            ->whereIn('status', [self::STATUS_ASSIGNED, self::STATUS_QUEUED])
            ->orderBy('created_at');
    }
}
