<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class FleetNode extends Model
{
    use BelongsToWorkspace;

    public const STATUS_OFFLINE = 'offline';

    public const STATUS_ONLINE = 'online';

    public const STATUS_BUSY = 'busy';

    public const STATUS_PAUSED = 'paused';

    protected $fillable = [
        'workspace_id',
        'agent_id',
        'platform',
        'models',
        'capabilities',
        'status',
        'compute_budget',
        'current_task_id',
        'last_heartbeat_at',
        'registered_at',
    ];

    protected $casts = [
        'models' => 'array',
        'capabilities' => 'array',
        'compute_budget' => 'array',
        'last_heartbeat_at' => 'datetime',
        'registered_at' => 'datetime',
    ];

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function currentTask(): BelongsTo
    {
        return $this->belongsTo(FleetTask::class, 'current_task_id');
    }

    public function tasks(): HasMany
    {
        return $this->hasMany(FleetTask::class);
    }

    public function creditEntries(): HasMany
    {
        return $this->hasMany(CreditEntry::class);
    }

    public function syncRecords(): HasMany
    {
        return $this->hasMany(SyncRecord::class);
    }

    public function scopeOnline(Builder $query): Builder
    {
        return $query->whereIn('status', [self::STATUS_ONLINE, self::STATUS_BUSY]);
    }

    public function scopeIdle(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_ONLINE)
            ->whereNull('current_task_id');
    }
}
