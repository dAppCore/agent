<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Mod\Content\Models\ContentTask;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Prompt extends Model
{
    use HasFactory;

    protected $fillable = [
        'name',
        'category',
        'description',
        'system_prompt',
        'user_template',
        'variables',
        'model',
        'model_config',
        'is_active',
    ];

    protected $casts = [
        'variables' => 'array',
        'model_config' => 'array',
        'is_active' => 'boolean',
    ];

    /**
     * Get the version history for this prompt.
     */
    public function versions(): HasMany
    {
        return $this->hasMany(PromptVersion::class)->orderByDesc('version');
    }

    /**
     * Get the content tasks using this prompt.
     */
    public function tasks(): HasMany
    {
        return $this->hasMany(ContentTask::class);
    }

    /**
     * Create a new version snapshot before saving changes.
     */
    public function createVersion(?int $userId = null): PromptVersion
    {
        $latestVersion = $this->versions()->max('version') ?? 0;

        return $this->versions()->create([
            'version' => $latestVersion + 1,
            'system_prompt' => $this->system_prompt,
            'user_template' => $this->user_template,
            'variables' => $this->variables,
            'created_by' => $userId,
        ]);
    }

    /**
     * Interpolate variables into the user template.
     */
    public function interpolate(array $data): string
    {
        $template = $this->user_template;

        foreach ($data as $key => $value) {
            if (is_string($value)) {
                $template = str_replace("{{{$key}}}", $value, $template);
            }
        }

        return $template;
    }

    /**
     * Scope to only active prompts.
     */
    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    /**
     * Scope by category.
     */
    public function scopeCategory(Builder $query, string $category): Builder
    {
        return $query->where('category', $category);
    }

    /**
     * Scope by model provider.
     */
    public function scopeForModel(Builder $query, string $model): Builder
    {
        return $query->where('model', $model);
    }
}
