<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * Plan Template Version - immutable snapshot of a YAML template's content.
 *
 * When a plan is created from a template, the template content is snapshotted
 * here so future edits to the YAML file do not affect existing plans.
 *
 * Identical content is deduplicated via content_hash so no duplicate rows
 * accumulate when the same (unchanged) template is used repeatedly.
 *
 * @property int $id
 * @property string $slug          Template file slug (filename without extension)
 * @property int $version          Sequential version number per slug
 * @property string $name          Template name at snapshot time
 * @property array $content        Full template content as JSON
 * @property string $content_hash  SHA-256 of json_encode($content)
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class PlanTemplateVersion extends Model
{
    protected $fillable = [
        'slug',
        'version',
        'name',
        'content',
        'content_hash',
    ];

    protected $casts = [
        'content' => 'array',
        'version' => 'integer',
    ];

    /**
     * Plans that were created from this template version.
     */
    public function plans(): HasMany
    {
        return $this->hasMany(AgentPlan::class, 'template_version_id');
    }

    /**
     * Find an existing version by content hash, or create a new one.
     *
     * Deduplicates identical template content so we don't store redundant rows
     * when the same (unchanged) template is used multiple times.
     */
    public static function findOrCreateFromTemplate(string $slug, array $content): self
    {
        $hash = hash('sha256', json_encode($content, JSON_UNESCAPED_UNICODE));

        $existing = static::where('slug', $slug)
            ->where('content_hash', $hash)
            ->first();

        if ($existing) {
            return $existing;
        }

        $nextVersion = (static::where('slug', $slug)->max('version') ?? 0) + 1;

        return static::create([
            'slug' => $slug,
            'version' => $nextVersion,
            'name' => $content['name'] ?? $slug,
            'content' => $content,
            'content_hash' => $hash,
        ]);
    }

    /**
     * Get all recorded versions for a template slug, newest first.
     *
     * @return Collection<int, static>
     */
    public static function historyFor(string $slug): Collection
    {
        return static::where('slug', $slug)
            ->orderByDesc('version')
            ->get();
    }
}
