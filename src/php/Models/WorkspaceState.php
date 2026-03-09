<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * Workspace State Model
 *
 * Persistent key-value state storage for agent plans.
 * Stores typed values shared across agent sessions within a plan,
 * enabling context sharing and state recovery.
 *
 * @property int $id
 * @property int $agent_plan_id
 * @property string $key
 * @property array $value
 * @property string $type
 * @property string|null $description
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class WorkspaceState extends Model
{
    use HasFactory;

    protected $table = 'agent_workspace_states';

    public const TYPE_JSON = 'json';

    public const TYPE_MARKDOWN = 'markdown';

    public const TYPE_CODE = 'code';

    public const TYPE_REFERENCE = 'reference';

    protected $fillable = [
        'agent_plan_id',
        'key',
        'value',
        'type',
        'description',
    ];

    protected $casts = [
        'value' => 'array',
    ];

    // Relationships

    public function plan(): BelongsTo
    {
        return $this->belongsTo(AgentPlan::class, 'agent_plan_id');
    }

    // Scopes

    public function scopeForPlan($query, AgentPlan|int $plan): mixed
    {
        $planId = $plan instanceof AgentPlan ? $plan->id : $plan;

<<<<<<< HEAD
    /**
     * Set typed value.
     */
    public function setTypedValue(mixed $value): void
    {
        $storedValue = match ($this->type) {
            self::TYPE_JSON => json_encode($value),
            default => (string) $value,
        };

        $this->update(['value' => $storedValue]);
    }

    /**
     * Get or create state for a plan.
     */
    public static function getOrCreate(AgentPlan $plan, string $key, mixed $default = null, string $type = self::TYPE_JSON): self
    {
        $state = static::where('agent_plan_id', $plan->id)
            ->where('key', $key)
            ->first();

        if (! $state) {
            $value = match ($type) {
                self::TYPE_JSON => json_encode($default),
                default => (string) ($default ?? ''),
            };

            $state = static::create([
                'agent_plan_id' => $plan->id,
                'key' => $key,
                'value' => $value,
                'type' => $type,
            ]);
        }

        return $state;
    }

    /**
     * Set state value for a plan.
     */
    public static function set(AgentPlan $plan, string $key, mixed $value, string $type = self::TYPE_JSON): self
    {
        $storedValue = match ($type) {
            self::TYPE_JSON => json_encode($value),
            default => (string) $value,
        };

        return static::updateOrCreate(
            ['agent_plan_id' => $plan->id, 'key' => $key],
            ['value' => $storedValue, 'type' => $type]
        );
    }

    /**
     * Get state value for a plan.
     */
    public static function get(AgentPlan $plan, string $key, mixed $default = null): mixed
    {
        $state = static::where('agent_plan_id', $plan->id)
            ->where('key', $key)
            ->first();

        if (! $state) {
            return $default;
        }

        return $state->getTypedValue();
    }

    /**
     * Scope: for plan.
     */
    public function scopeForPlan(Builder $query, int $planId): Builder
    {
        return $query->where('agent_plan_id', $planId);
    }

    /**
     * Scope: by type.
     */
    public function scopeByType(Builder $query, string $type): Builder
    {
        return $query->where('type', $type);
    }

    // Type helpers

    public function isJson(): bool
    {
        return $this->type === self::TYPE_JSON;
    }

    public function isMarkdown(): bool
    {
        return $this->type === self::TYPE_MARKDOWN;
    }

    public function isCode(): bool
    {
        return $this->type === self::TYPE_CODE;
    }

    public function isReference(): bool
    {
        return $this->type === self::TYPE_REFERENCE;
    }

    public function getFormattedValue(): string
    {
        if ($this->isMarkdown() || $this->isCode()) {
            return is_string($this->value) ? $this->value : json_encode($this->value, JSON_PRETTY_PRINT);
        }

        return json_encode($this->value, JSON_PRETTY_PRINT);
    }

    // Static helpers

    /**
     * Get a state value for a plan, returning $default if not set.
     */
    public static function getValue(AgentPlan $plan, string $key, mixed $default = null): mixed
    {
        $state = static::where('agent_plan_id', $plan->id)->where('key', $key)->first();

        return $state !== null ? $state->value : $default;
    }

    /**
     * Set (upsert) a state value for a plan.
     */
    public static function setValue(AgentPlan $plan, string $key, mixed $value, string $type = self::TYPE_JSON): self
    {
        return static::updateOrCreate(
            ['agent_plan_id' => $plan->id, 'key' => $key],
            ['value' => $value, 'type' => $type]
        );
    }

    // MCP output

    public function toMcpContext(): array
    {
        return [
            'key' => $this->key,
            'type' => $this->type,
            'description' => $this->description,
            'value' => $this->value,
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }
}
