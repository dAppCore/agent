<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class CreditEntry extends Model
{
    use BelongsToWorkspace;

    protected $fillable = [
        'workspace_id',
        'fleet_node_id',
        'task_type',
        'amount',
        'balance_after',
        'description',
    ];

    protected $casts = [
        'amount' => 'integer',
        'balance_after' => 'integer',
    ];

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function fleetNode(): BelongsTo
    {
        return $this->belongsTo(FleetNode::class);
    }
}
