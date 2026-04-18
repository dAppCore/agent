<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class SyncRecord extends Model
{
    protected $fillable = [
        'fleet_node_id',
        'direction',
        'payload_size',
        'items_count',
        'synced_at',
    ];

    protected $casts = [
        'payload_size' => 'integer',
        'items_count' => 'integer',
        'synced_at' => 'datetime',
    ];

    public function fleetNode(): BelongsTo
    {
        return $this->belongsTo(FleetNode::class);
    }
}
