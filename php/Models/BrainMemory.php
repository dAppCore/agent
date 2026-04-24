<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Carbon\Carbon;
use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;

/**
 * Brain Memory - a unit of shared knowledge in the OpenBrain store.
 *
 * Agents write observations, decisions, conventions, and research
 * into the brain so that other agents (and future sessions) can
 * recall organisational knowledge without re-discovering it.
 *
 * @property string $id
 * @property int $workspace_id
 * @property string $agent_id
 * @property string $type
 * @property string $content
 * @property array|null $tags
 * @property string|null $org
 * @property string|null $project
 * @property float $confidence
 * @property string|null $supersedes_id
 * @property Carbon|null $indexed_at
 * @property Carbon|null $expires_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property Carbon|null $deleted_at
 */
class BrainMemory extends Model
{
    use BelongsToWorkspace;
    use HasUuids;
    use SoftDeletes;

    /** Valid memory types. */
    public const VALID_TYPES = [
        'fact',
        'decision',
        'observation',
        'convention',
        'research',
        'plan',
        'bug',
        'architecture',
        'documentation',
        'service',
        'pattern',
        'context',
        'procedure',
    ];

    protected $connection = 'brain';

    protected $table = 'brain_memories';

    protected $fillable = [
        'workspace_id',
        'agent_id',
        'type',
        'content',
        'tags',
        'org',
        'project',
        'confidence',
        'supersedes_id',
        'indexed_at',
        'expires_at',
        'source',
    ];

    protected $casts = [
        'tags' => 'array',
        'confidence' => 'float',
        'indexed_at' => 'datetime',
        'expires_at' => 'datetime',
    ];

    // ----------------------------------------------------------------
    // Relationships
    // ----------------------------------------------------------------

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    /** The older memory this one replaces. */
    public function supersedes(): BelongsTo
    {
        return $this->belongsTo(self::class, 'supersedes_id');
    }

    /** Newer memories that replaced this one. */
    public function supersededBy(): HasMany
    {
        return $this->hasMany(self::class, 'supersedes_id');
    }

    // ----------------------------------------------------------------
    // Scopes
    // ----------------------------------------------------------------

    public function scopeForWorkspace(Builder $query, int $workspaceId): Builder
    {
        return $query->where('workspace_id', $workspaceId);
    }

    public function scopeOfType(Builder $query, string|array $type): Builder
    {
        return is_array($type)
            ? $query->whereIn('type', $type)
            : $query->where('type', $type);
    }

    public function scopeForProject(Builder $query, ?string $project): Builder
    {
        return $project
            ? $query->where('project', $project)
            : $query;
    }

    public function scopeForOrg(Builder $query, ?string $org): Builder
    {
        return $org
            ? $query->where('org', $org)
            : $query;
    }

    public function scopeByAgent(Builder $query, ?string $agentId): Builder
    {
        return $agentId
            ? $query->where('agent_id', $agentId)
            : $query;
    }

    /** Exclude memories whose TTL has passed. */
    public function scopeActive(Builder $query): Builder
    {
        return $query->where(function (Builder $q) {
            $q->whereNull('expires_at')
                ->orWhere('expires_at', '>', now());
        });
    }

    /** Exclude memories that have been superseded by a newer version. */
    public function scopeLatestVersions(Builder $query): Builder
    {
        return $query->whereDoesntHave('supersededBy', function (Builder $q) {
            $q->whereNull('deleted_at');
        });
    }

    // ----------------------------------------------------------------
    // Helpers
    // ----------------------------------------------------------------

    /**
     * Walk the supersession chain and return its depth.
     *
     * A memory that supersedes nothing returns 0.
     * Capped at 50 to prevent runaway loops.
     */
    public function getSupersessionDepth(): int
    {
        $depth = 0;
        $current = $this;
        $maxDepth = 50;

        while ($current->supersedes_id !== null && $depth < $maxDepth) {
            $current = self::withTrashed()->find($current->supersedes_id);

            if ($current === null) {
                break;
            }

            $depth++;
        }

        return $depth;
    }

    /** Format the memory for MCP tool responses. */
    public function toMcpContext(float $score = 0.0): array
    {
        return [
            'id' => $this->id,
            'agent_id' => $this->agent_id,
            'type' => $this->type,
            'content' => $this->content,
            'tags' => $this->tags ?? [],
            'org' => $this->getAttribute('org'),
            'project' => $this->project,
            'confidence' => $this->confidence,
            'score' => round($score, 4),
            'source' => $this->source ?? 'manual',
            'supersedes_id' => $this->supersedes_id,
            'supersedes_count' => $this->getSupersessionDepth(),
            'expires_at' => $this->expires_at?->toIso8601String(),
            'deleted_at' => $this->deleted_at?->toIso8601String(),
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }
}
