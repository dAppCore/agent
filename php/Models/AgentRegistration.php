<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class AgentRegistration extends Model
{
    use BelongsToWorkspace;
    use HasUuids;

    public $incrementing = false;

    protected $keyType = 'string';

    protected $fillable = [
        'workspace_id',
        'agent_id',
        'hostname',
        'platform',
        'capabilities',
        'models',
        'compute_budget',
        'max_concurrent',
        'labels',
        'version',
        'status',
        'current_task_id',
        'connected_at',
        'last_heartbeat_at',
        'metadata',
    ];

    protected $casts = [
        'capabilities' => 'array',
        'models' => 'array',
        'compute_budget' => 'array',
        'labels' => 'array',
        'max_concurrent' => 'integer',
        'connected_at' => 'datetime',
        'last_heartbeat_at' => 'datetime',
        'metadata' => 'array',
    ];

    public const STATUS_ONLINE = 'online';

    public const STATUS_OFFLINE = 'offline';

    public const STATUS_PAUSED = 'paused';

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function scopeOnline(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_ONLINE)
            ->where('last_heartbeat_at', '>=', now()->subMinutes(5));
    }

    public function hasCapability(?string $agentType): bool
    {
        if ($agentType === null || $agentType === '') {
            return true;
        }

        return in_array($agentType, $this->capabilities ?? [], true);
    }

    /**
     * @param  array<int, string>|null  $requiredLabels
     */
    public function hasLabels(?array $requiredLabels): bool
    {
        if ($requiredLabels === null || $requiredLabels === []) {
            return true;
        }

        $labels = $this->labels ?? [];

        foreach ($requiredLabels as $label) {
            if (! in_array($label, $labels, true)) {
                return false;
            }
        }

        return true;
    }
}
