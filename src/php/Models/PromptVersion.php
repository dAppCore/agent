<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Models\User;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class PromptVersion extends Model
{
    protected $fillable = [
        'prompt_id',
        'version',
        'system_prompt',
        'user_template',
        'variables',
        'created_by',
    ];

    protected $casts = [
        'variables' => 'array',
        'version' => 'integer',
    ];

    /**
     * Get the parent prompt.
     */
    public function prompt(): BelongsTo
    {
        return $this->belongsTo(Prompt::class);
    }

    /**
     * Get the user who created this version.
     */
    public function creator(): BelongsTo
    {
        return $this->belongsTo(User::class, 'created_by');
    }

    /**
     * Restore this version to the parent prompt.
     */
    public function restore(): Prompt
    {
        $this->prompt->update([
            'system_prompt' => $this->system_prompt,
            'user_template' => $this->user_template,
            'variables' => $this->variables,
        ]);

        return $this->prompt;
    }
}
