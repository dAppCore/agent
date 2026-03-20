<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

class Task extends Model
{
    use BelongsToWorkspace;

    protected $fillable = [
        'workspace_id',
        'title',
        'description',
        'status',
        'priority',
        'category',
        'file_ref',
        'line_ref',
    ];

    protected $casts = [
        'line_ref' => 'integer',
    ];

    public function scopePending(Builder $query): Builder
    {
        return $query->where('status', 'pending');
    }

    public function scopeInProgress(Builder $query): Builder
    {
        return $query->where('status', 'in_progress');
    }

    public function scopeDone(Builder $query): Builder
    {
        return $query->where('status', 'done');
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->whereIn('status', ['pending', 'in_progress']);
    }

    /**
     * Order by priority using CASE statement with whitelisted values.
     *
     * This is a safe replacement for orderByRaw("FIELD(priority, ...)") which
     * could be vulnerable to SQL injection if extended with user input.
     */
    public function scopeOrderByPriority($query, string $direction = 'asc')
    {
        return $query->orderByRaw('CASE priority
            WHEN ? THEN 1
            WHEN ? THEN 2
            WHEN ? THEN 3
            WHEN ? THEN 4
            ELSE 5
        END '.($direction === 'desc' ? 'DESC' : 'ASC'), ['urgent', 'high', 'normal', 'low']);
    }

    /**
     * Order by status using CASE statement with whitelisted values.
     *
     * This is a safe replacement for orderByRaw("FIELD(status, ...)") which
     * could be vulnerable to SQL injection if extended with user input.
     */
    public function scopeOrderByStatus($query, string $direction = 'asc')
    {
        return $query->orderByRaw('CASE status
            WHEN ? THEN 1
            WHEN ? THEN 2
            WHEN ? THEN 3
            ELSE 4
        END '.($direction === 'desc' ? 'DESC' : 'ASC'), ['in_progress', 'pending', 'done']);
    }

    public function getStatusBadgeAttribute(): string
    {
        return match ($this->status) {
            'done' => '✓',
            'in_progress' => '→',
            default => '○',
        };
    }

    public function getPriorityBadgeAttribute(): string
    {
        return match ($this->priority) {
            'urgent' => '🔴',
            'high' => '🟠',
            'low' => '🔵',
            default => '',
        };
    }
}
