<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Plan Retention Policy
    |--------------------------------------------------------------------------
    |
    | Archived plans are permanently deleted after this many days. This frees
    | up storage and keeps the database lean over time.
    |
    | Set to 0 or null to disable automatic cleanup entirely.
    |
    | Default: 90 days
    |
    */

    'plan_retention_days' => env('AGENTIC_PLAN_RETENTION_DAYS', 90),

    /*
    |--------------------------------------------------------------------------
    | Forgejo Integration
    |--------------------------------------------------------------------------
    |
    | Configuration for the Forgejo-based scan/dispatch/PR pipeline.
    | AGENTIC_SCAN_REPOS is a comma-separated list of owner/name repos.
    |
    */

    'scan_repos' => array_filter(explode(',', env('AGENTIC_SCAN_REPOS', ''))),

    'forge_url' => env('FORGE_URL', 'https://forge.lthn.ai'),

    'forge_token' => env('FORGE_TOKEN', ''),

];
