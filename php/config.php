<?php

return [

    /*
    |--------------------------------------------------------------------------
    | MCP Portal Domain
    |--------------------------------------------------------------------------
    |
    | Default domain for the MCP Portal. The app-level Boot may override this
    | with a wildcard (e.g. mcp.{tld}) for multi-domain support.
    |
    */

    'domain' => env('MCP_DOMAIN', 'mcp.'.env('APP_DOMAIN', 'host.uk.com')),

    /*
    |--------------------------------------------------------------------------
    | Registry Path
    |--------------------------------------------------------------------------
    |
    | Where to find MCP server definitions. Each server has its own YAML file
    | in the servers subdirectory.
    |
    */

    'registry_path' => resource_path('mcp'),

    /*
    |--------------------------------------------------------------------------
    | Plan Templates Path
    |--------------------------------------------------------------------------
    |
    | Where agent plan templates are stored. These define structured workflows
    | for common development tasks.
    |
    */

    'templates_path' => resource_path('plan-templates'),

    /*
    |--------------------------------------------------------------------------
    | Content Generation Paths
    |--------------------------------------------------------------------------
    |
    | Paths for the ContentService batch generation system.
    |
    */

    'content' => [
        'batch_path' => 'app/Mod/Agentic/Resources/tasks',
        'prompt_path' => 'app/Mod/Agentic/Resources/prompts/content',
        'drafts_path' => 'app/Mod/Agentic/Resources/drafts',
    ],

    /*
    |--------------------------------------------------------------------------
    | Cache Keys
    |--------------------------------------------------------------------------
    |
    | Namespaced cache keys used by agentic endpoints. Override these in your
    | application config to prevent collisions with other modules.
    |
    */

    'cache' => [
        'for_agents_key' => 'agentic.for-agents.json',
        'for_agents_ttl' => 3600,
    ],

    /*
    |--------------------------------------------------------------------------
    | OpenBrain (Shared Agent Knowledge Store)
    |--------------------------------------------------------------------------
    |
    | Configuration for the vector-indexed knowledge store. Requires
    | Ollama (for embeddings) and Qdrant (for vector search).
    |
    */

    'brain' => [
        'ollama_url' => env('BRAIN_OLLAMA_URL', 'https://ollama.lthn.sh'),
        'qdrant_url' => env('BRAIN_QDRANT_URL', 'https://qdrant.lthn.sh'),
        'qdrant' => [
            'api_key' => env('BRAIN_QDRANT_API_KEY', ''),
        ],
        'collection' => env('BRAIN_COLLECTION', 'openbrain'),
        'embedding_model' => env('BRAIN_EMBEDDING_MODEL', 'embeddinggemma'),

        // Dedicated database connection for brain_memories.
        // Defaults to the app's main database when BRAIN_DB_* env vars are absent.
        // Set BRAIN_DB_HOST to a remote MariaDB (e.g. the homelab) to co-locate
        // DB rows with their Qdrant vectors.
        'database' => [
            'driver' => env('BRAIN_DB_DRIVER', env('DB_CONNECTION', 'mariadb')),
            'host' => env('BRAIN_DB_HOST', env('DB_HOST', '127.0.0.1')),
            'port' => env('BRAIN_DB_PORT', env('DB_PORT', '3306')),
            'database' => env('BRAIN_DB_DATABASE', env('DB_DATABASE', 'forge')),
            'username' => env('BRAIN_DB_USERNAME', env('DB_USERNAME', 'forge')),
            'password' => env('BRAIN_DB_PASSWORD', env('DB_PASSWORD', '')),
            // charset defaults: utf8 for pgsql (Postgres rejects 'utf8mb4'),
            // utf8mb4 for everything else (MariaDB/MySQL). Override with
            // BRAIN_DB_CHARSET if your instance needs something specific.
            'charset' => env('BRAIN_DB_CHARSET', env('BRAIN_DB_DRIVER', env('DB_CONNECTION', 'mariadb')) === 'pgsql' ? 'utf8' : 'utf8mb4'),
            'collation' => env('BRAIN_DB_COLLATION', 'utf8mb4_unicode_ci'),
            'prefix' => '',
        ],
    ],

];
