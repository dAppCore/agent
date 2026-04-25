<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

class AgentProfile extends Model
{
    protected $fillable = [
        'name',
        'gateway_url',
        'api_key_cipher',
        'cost_class',
        'capability_tags',
        'quota_headroom_pct',
        'enabled',
        'last_dispatched_at',
    ];

    protected $casts = [
        'api_key_cipher' => 'encrypted',
        'capability_tags' => 'array',
        'quota_headroom_pct' => 'integer',
        'enabled' => 'boolean',
        'last_dispatched_at' => 'datetime',
    ];

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('enabled', true)
            ->where('quota_headroom_pct', '>', 0);
    }

    public function scopeForClass(Builder $query, string $class): Builder
    {
        return $query->where('cost_class', $class);
    }
}
