<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * AgentMessage — direct messages between agents.
 *
 * Not semantic, not vector-searched. Just chronological messages.
 */
class AgentMessage extends Model
{
    use BelongsToWorkspace;

    protected $fillable = [
        'workspace_id',
        'from_agent',
        'to_agent',
        'content',
        'subject',
        'read_at',
    ];

    protected $casts = [
        'read_at' => 'datetime',
    ];

    public function scopeInbox(Builder $query, string $agent): Builder
    {
        return $query->where('to_agent', $agent)->orderByDesc('created_at');
    }

    public function scopeUnread(Builder $query): Builder
    {
        return $query->whereNull('read_at');
    }

    public function scopeConversation(Builder $query, string $agent1, string $agent2): Builder
    {
        return $query->where(function ($q) use ($agent1, $agent2) {
            $q->where(function ($q2) use ($agent1, $agent2) {
                $q2->where('from_agent', $agent1)->where('to_agent', $agent2);
            })->orWhere(function ($q2) use ($agent1, $agent2) {
                $q2->where('from_agent', $agent2)->where('to_agent', $agent1);
            });
        })->orderByDesc('created_at');
    }

    public function markRead(): void
    {
        if (! $this->read_at) {
            $this->update(['read_at' => now()]);
        }
    }
}
