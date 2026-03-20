<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Service;

use Core\Events\AdminPanelBooting;
use Core\Front\Admin\AdminMenuRegistry;
use Core\Service\Contracts\ServiceDefinition;
use Core\Service\ServiceVersion;
use Illuminate\Support\ServiceProvider;

/**
 * Agentic Service
 *
 * AI agent orchestration service layer.
 * Uses Core\Agentic as the engine.
 */
class Boot extends ServiceProvider implements ServiceDefinition
{
    /**
     * Events this service listens to.
     *
     * @var array<class-string, string>
     */
    public static array $listens = [
        AdminPanelBooting::class => 'onAdminPanel',
    ];

    /**
     * Bootstrap the service.
     */
    public function boot(): void
    {
        app(AdminMenuRegistry::class)->register($this);
    }

    /**
     * Get the service definition for seeding platform_services.
     */
    public static function definition(): array
    {
        return [
            'code' => 'agentic',
            'module' => 'Agentic',
            'name' => 'Agentic',
            'tagline' => 'AI agent orchestration',
            'description' => 'Build and deploy AI agents with planning, tool use, and conversation capabilities.',
            'icon' => 'robot',
            'color' => 'violet',
            'marketing_domain' => null, // API service, no marketing site yet
            'website_class' => null,
            'entitlement_code' => 'core.srv.agentic',
            'sort_order' => 60,
        ];
    }

    /**
     * Admin menu items for this service.
     *
     * Agentic has its own top-level group right after Dashboard —
     * this is the primary capability of the platform.
     */
    public function adminMenuItems(): array
    {
        return [
            [
                'group' => 'agents',
                'priority' => 5,
                'entitlement' => 'core.srv.agentic',
                'item' => fn () => [
                    'label' => 'Agentic',
                    'icon' => 'robot',
                    'color' => 'violet',
                    'active' => request()->routeIs('hub.agents.*'),
                    'children' => [
                        ['label' => 'Dashboard', 'icon' => 'gauge', 'href' => route('hub.agents.index'), 'active' => request()->routeIs('hub.agents.index')],
                        ['label' => 'Plans', 'icon' => 'list-check', 'href' => route('hub.agents.plans'), 'active' => request()->routeIs('hub.agents.plans*')],
                        ['label' => 'Sessions', 'icon' => 'messages', 'href' => route('hub.agents.sessions'), 'active' => request()->routeIs('hub.agents.sessions*')],
                        ['label' => 'Tool Analytics', 'icon' => 'chart-bar', 'href' => route('hub.agents.tools'), 'active' => request()->routeIs('hub.agents.tools*')],
                        ['label' => 'API Keys', 'icon' => 'key', 'href' => route('hub.agents.api-keys'), 'active' => request()->routeIs('hub.agents.api-keys*')],
                        ['label' => 'Templates', 'icon' => 'copy', 'href' => route('hub.agents.templates'), 'active' => request()->routeIs('hub.agents.templates*')],
                    ],
                ],
            ],
        ];
    }

    /**
     * Register admin panel components.
     */
    public function onAdminPanel(AdminPanelBooting $event): void
    {
        // Service-specific admin routes could go here
        // Components are registered by Core\Agentic
    }

    public function menuPermissions(): array
    {
        return [];
    }

    public function canViewMenu(?object $user, ?object $workspace): bool
    {
        return $user !== null;
    }

    public static function version(): ServiceVersion
    {
        return new ServiceVersion(1, 0, 0);
    }

    /**
     * Service dependencies.
     */
    public static function dependencies(): array
    {
        return [];
    }
}
