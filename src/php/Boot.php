<?php

declare(strict_types=1);

namespace Core\Mod\Agentic;

use Core\Events\AdminPanelBooting;
use Core\Events\ApiRoutesRegistering;
use Core\Events\ConsoleBooting;
use Core\Events\McpToolsRegistering;
use Illuminate\Cache\RateLimiting\Limit;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\RateLimiter;
use Illuminate\Support\ServiceProvider;

class Boot extends ServiceProvider
{
    protected string $moduleName = 'agentic';

    /**
     * Events this module listens to for lazy loading.
     *
     * @var array<class-string, string>
     */
    public static array $listens = [
        AdminPanelBooting::class => 'onAdminPanel',
        ApiRoutesRegistering::class => 'onApiRoutes',
        ConsoleBooting::class => 'onConsole',
        McpToolsRegistering::class => 'onMcpTools',
    ];

    public function boot(): void
    {
        $this->loadMigrationsFrom(__DIR__.'/Migrations');
        $this->loadTranslationsFrom(__DIR__.'/Lang', 'agentic');
        $this->configureRateLimiting();
        $this->scheduleCommands();
    }

    /**
     * Register all scheduled commands.
     */
    protected function scheduleCommands(): void
    {
        $this->app->booted(function (): void {
            $schedule = $this->app->make(Schedule::class);
            $schedule->command('agentic:plan-cleanup')->daily();

            // Forgejo pipeline — only active when a token is configured
            if (config('agentic.forge_token') !== '') {
                $schedule->command('agentic:scan')->everyFiveMinutes();
                $schedule->command('agentic:dispatch')->everyTwoMinutes();
                $schedule->command('agentic:pr-manage')->everyFiveMinutes();
            }
        });
    }

    /**
     * Configure rate limiting for agentic endpoints.
     */
    protected function configureRateLimiting(): void
    {
        // Rate limit for the for-agents.json endpoint
        // Allow 60 requests per minute per IP
        RateLimiter::for('agentic-api', function (Request $request) {
            return Limit::perMinute(60)->by($request->ip());
        });
    }

    public function register(): void
    {
        $this->mergeConfigFrom(
            __DIR__.'/config.php',
            'mcp'
        );

        $this->mergeConfigFrom(
            __DIR__.'/agentic.php',
            'agentic'
        );

        // Register the dedicated brain database connection.
        // Falls back to the app's default DB when no BRAIN_DB_* env vars are set.
        $brainDb = config('mcp.brain.database');

        if (is_array($brainDb) && ! empty($brainDb['host'])) {
            config(['database.connections.brain' => $brainDb]);
        }

        $this->app->singleton(\Core\Mod\Agentic\Services\AgenticManager::class);
        $this->app->singleton(\Core\Mod\Agentic\Services\AgentToolRegistry::class);

        $this->app->singleton(\Core\Mod\Agentic\Services\ForgejoService::class, function ($app) {
            return new \Core\Mod\Agentic\Services\ForgejoService(
                baseUrl: (string) config('agentic.forge_url', 'https://forge.lthn.ai'),
                token: (string) config('agentic.forge_token', ''),
            );
        });

        $this->app->singleton(\Core\Mod\Agentic\Services\BrainService::class, function ($app) {
            $ollamaUrl = config('mcp.brain.ollama_url', 'http://localhost:11434');
            $qdrantUrl = config('mcp.brain.qdrant_url', 'http://localhost:6334');

            // Skip TLS verification for non-public TLDs (self-signed certs behind Traefik)
            $hasLocalTld = static fn (string $url): bool => (bool) preg_match(
                '/\.(lan|lab|local|test)(?:[:\/]|$)/',
                parse_url($url, PHP_URL_HOST) ?? ''
            );
            $verifySsl = ! ($hasLocalTld($ollamaUrl) || $hasLocalTld($qdrantUrl));

            return new \Core\Mod\Agentic\Services\BrainService(
                ollamaUrl: $ollamaUrl,
                qdrantUrl: $qdrantUrl,
                collection: config('mcp.brain.collection', 'openbrain'),
                embeddingModel: config('mcp.brain.embedding_model', 'nomic-embed-text'),
                verifySsl: $verifySsl,
            );
        });
    }

    // -------------------------------------------------------------------------
    // Event-driven handlers (for lazy loading once event system is integrated)
    // -------------------------------------------------------------------------

    /**
     * Handle API routes registration event.
     *
     * Registers REST API endpoints for go-agentic Client consumption.
     * Routes at /v1/* — Go client uses BaseURL + "/v1/plans" directly.
     */
    public function onApiRoutes(ApiRoutesRegistering $event): void
    {
        // Register agent API auth middleware alias
        $event->middleware('agent.auth', Middleware\AgentApiAuth::class);

        // Scoped via event — only loaded for API requests
        if (file_exists(__DIR__.'/Routes/api.php')) {
            $event->routes(fn () => require __DIR__.'/Routes/api.php');
        }
    }

    /**
     * Handle admin panel booting event.
     */
    public function onAdminPanel(AdminPanelBooting $event): void
    {
        $event->views($this->moduleName, __DIR__.'/View/Blade');

        // Register admin routes
        if (file_exists(__DIR__.'/Routes/admin.php')) {
            $event->routes(fn () => require __DIR__.'/Routes/admin.php');
        }

        // Register Livewire components
        $event->livewire('agentic.admin.dashboard', View\Modal\Admin\Dashboard::class);
        $event->livewire('agentic.admin.plans', View\Modal\Admin\Plans::class);
        $event->livewire('agentic.admin.plan-detail', View\Modal\Admin\PlanDetail::class);
        $event->livewire('agentic.admin.sessions', View\Modal\Admin\Sessions::class);
        $event->livewire('agentic.admin.session-detail', View\Modal\Admin\SessionDetail::class);
        $event->livewire('agentic.admin.tool-analytics', View\Modal\Admin\ToolAnalytics::class);
        $event->livewire('agentic.admin.tool-calls', View\Modal\Admin\ToolCalls::class);
        $event->livewire('agentic.admin.api-keys', View\Modal\Admin\ApiKeys::class);
        $event->livewire('agentic.admin.templates', View\Modal\Admin\Templates::class);

        // Note: Navigation is registered via AdminMenuProvider interface
        // in the existing boot() method until we migrate to pure event-driven nav.
    }

    /**
     * Handle API routes registration event.
     */
    public function onApiRoutes(ApiRoutesRegistering $event): void
    {
        $event->routes(fn () => require __DIR__.'/Routes/api.php');
    }

    /**
     * Handle console booting event.
     */
    public function onConsole(ConsoleBooting $event): void
    {
        // Register middleware alias for CLI context (artisan route:list)
        $event->middleware('agent.auth', Middleware\AgentApiAuth::class);

        $event->command(Console\Commands\TaskCommand::class);
        $event->command(Console\Commands\PlanCommand::class);
        $event->command(Console\Commands\GenerateCommand::class);
        $event->command(Console\Commands\PlanRetentionCommand::class);
        $event->command(Console\Commands\BrainSeedMemoryCommand::class);
        $event->command(Console\Commands\BrainIngestCommand::class);
        $event->command(Console\Commands\ScanCommand::class);
        $event->command(Console\Commands\DispatchCommand::class);
        $event->command(Console\Commands\PrManageCommand::class);
        $event->command(Console\Commands\PrepWorkspaceCommand::class);
    }

    /**
     * Handle MCP tools registration event.
     *
     * Note: Agent tools (plan_create, session_start, etc.) are implemented in
     * the Mcp module at Mod\Mcp\Tools\Agent\* and registered via AgentToolRegistry.
     * Brain tools are registered here as they belong to the Agentic module.
     */
    public function onMcpTools(McpToolsRegistering $event): void
    {
        $registry = $this->app->make(Services\AgentToolRegistry::class);

        $registry->registerMany([
            new Mcp\Tools\Agent\Brain\BrainRemember(),
            new Mcp\Tools\Agent\Brain\BrainRecall(),
            new Mcp\Tools\Agent\Brain\BrainForget(),
            new Mcp\Tools\Agent\Brain\BrainList(),
        ]);
    }
}
