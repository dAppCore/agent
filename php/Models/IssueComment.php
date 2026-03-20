<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * IssueComment — a comment on an issue.
 *
 * @property int $id
 * @property int $issue_id
 * @property string $author
 * @property string $body
 * @property array|null $metadata
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class IssueComment extends Model
{
    protected $table = 'issue_comments';

    protected $fillable = [
        'issue_id',
        'author',
        'body',
        'metadata',
    ];

    protected $casts = [
        'metadata' => 'array',
    ];

    public function issue(): BelongsTo
    {
        return $this->belongsTo(Issue::class);
    }

    public function toMcpContext(): array
    {
        return [
            'id' => $this->id,
            'author' => $this->author,
            'body' => $this->body,
            'metadata' => $this->metadata,
            'created_at' => $this->created_at?->toIso8601String(),
        ];
    }
}
