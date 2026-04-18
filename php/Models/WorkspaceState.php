<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class WorkspaceState extends Model
{
    use HasFactory;

    protected $table = 'agent_workspace_states';

    public const CATEGORY_GENERAL = 'general';

    public const TYPE_JSON = 'json';

    public const TYPE_MARKDOWN = 'markdown';

    public const TYPE_CODE = 'code';

    public const TYPE_REFERENCE = 'reference';

    protected $fillable = [
        'agent_plan_id',
        'key',
        'category',
        'value',
        'type',
        'description',
    ];

    protected $casts = [
        'value' => 'array',
    ];

    public function plan(): BelongsTo
    {
        return $this->belongsTo(AgentPlan::class, 'agent_plan_id');
    }

    public function scopeForPlan(Builder $query, AgentPlan|int $plan): Builder
    {
        $planId = $plan instanceof AgentPlan ? $plan->id : $plan;

        return $query->where('agent_plan_id', $planId);
    }

    public function scopeOfType(Builder $query, string $type): Builder
    {
        return $query->where('type', $type);
    }

    public function scopeInCategory(Builder $query, string $category): Builder
    {
        return $query->where('category', $category);
    }

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
        $value = $this->value;

        if (is_string($value)) {
            return $value;
        }

        return (string) json_encode($value, JSON_PRETTY_PRINT);
    }

    public static function getOrCreate(
        AgentPlan|int $plan,
        string $key,
        mixed $default = null,
        string $type = self::TYPE_JSON
    ): self {
        $planId = static::planId($plan);

        return static::firstOrCreate(
            ['agent_plan_id' => $planId, 'key' => $key],
            ['value' => $default, 'type' => $type]
        );
    }

    public static function getValue(AgentPlan|int $plan, string $key, mixed $default = null): mixed
    {
        $state = static::forPlan($plan)->where('key', $key)->first();

        return $state?->value ?? $default;
    }

    public static function setValue(
        AgentPlan|int $plan,
        string $key,
        mixed $value,
        string $type = self::TYPE_JSON
    ): self {
        $planId = static::planId($plan);

        return static::updateOrCreate(
            ['agent_plan_id' => $planId, 'key' => $key],
            ['value' => $value, 'type' => $type]
        );
    }

    /**
     * WorkspaceState::set($plan->id, 'discovered_pattern', 'observer');
     * WorkspaceState::get($plan->id, 'discovered_pattern');
     */
    public static function set(
        AgentPlan|int $plan,
        string $key,
        mixed $value,
        string $type = self::TYPE_JSON
    ): self {
        return static::setValue($plan, $key, $value, $type);
    }

    public static function get(AgentPlan|int $plan, string $key, mixed $default = null): mixed
    {
        return static::getValue($plan, $key, $default);
    }

    public function setTypedValue(mixed $value): void
    {
        $this->update(['value' => $value]);
    }

    public function getTypedValue(): mixed
    {
        return $this->value;
    }

    public function toMcpContext(): array
    {
        return [
            'key' => $this->key,
            'category' => $this->category ?? self::CATEGORY_GENERAL,
            'type' => $this->type,
            'description' => $this->description,
            'value' => $this->value,
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }

    private static function planId(AgentPlan|int $plan): int
    {
        return $plan instanceof AgentPlan ? $plan->id : $plan;
    }
}
