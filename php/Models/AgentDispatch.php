<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Model;

class AgentDispatch extends Model
{
    public const STATUS_QUEUED = 'queued';

    protected $fillable = [
        'ticket_id',
        'profile_id',
        'response_id',
        'run_id',
        'status',
        'error',
    ];
}
