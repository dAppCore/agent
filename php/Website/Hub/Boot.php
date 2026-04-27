<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub;

use Core\Events\AdminPanelBooting;
use Core\Events\DomainResolving;
use Core\Mod\Agentic\Mod\Admin\Menu\AdminMenuRegistry;
use Core\Mod\Agentic\Mod\Admin\Menu\Contracts\AdminMenuProvider;
use Core\Mod\Agentic\Website\Hub\Support\HubRouteNames;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\ServiceProvider;

class Boot extends ServiceProvider implements AdminMenuProvider
{
    /**
     * @var array<string>
     */
    public static array $domains = [
        '/^core\.(test|localhost)$/',
        '/^hub\.core\.(test|localhost)$/',
    ];

    /**
     * @var array<class-string, string>
     */
    public static array $listens = [
        DomainResolving::class => 'onDomainResolving',
        AdminPanelBooting::class => 'onAdminPanel',
    ];

    public function onDomainResolving(DomainResolving $event): void
    {
        foreach (static::$domains as $pattern) {
            if ($event->matches($pattern)) {
                $event->register(static::class);

                return;
            }
        }
    }

    public function onAdminPanel(AdminPanelBooting $event): void
    {
        $event->views('hub', __DIR__.'/View/Blade');
        $event->translations('hub', dirname(__DIR__, 2).'/Mod/Admin/Lang');
        $event->livewire('hub.admin.workspace-switcher', View\Modal\Admin\WorkspaceSwitcher::class);
        $event->livewire('hub.admin.global-search', View\Modal\Admin\GlobalSearch::class);

        app(AdminMenuRegistry::class)->register($this);

        $domain = request()->getHost();
        $routePrefix = HubRouteNames::prefix($domain);

        $event->routes(fn () => Route::prefix('hub')
            ->name($routePrefix)
            ->domain($domain)
            ->group(__DIR__.'/Routes/admin.php'));
    }

    public function adminMenuItems(): array
    {
        return [
            [
                'group' => 'dashboard',
                'priority' => 10,
                'item' => fn (): array => [
                    'label' => 'Dashboard',
                    'icon' => 'house',
                    'href' => HubRouteNames::url('dashboard', fallback: '/hub'),
                    'active' => request()->routeIs(HubRouteNames::name('dashboard')),
                ],
            ],
            [
                'group' => 'workspaces',
                'priority' => 10,
                'item' => fn (): array => [
                    'label' => 'Workspaces',
                    'icon' => 'folders',
                    'href' => HubRouteNames::url('sites', fallback: '/hub/workspaces'),
                    'active' => request()->routeIs(HubRouteNames::name('sites'))
                        || request()->routeIs(HubRouteNames::name('sites.settings')),
                ],
            ],
            [
                'group' => 'settings',
                'priority' => 30,
                'item' => fn (): array => [
                    'label' => 'Usage',
                    'icon' => 'chart-pie',
                    'href' => HubRouteNames::url('account.usage', fallback: '/hub/account/usage'),
                    'active' => request()->routeIs(HubRouteNames::name('account.usage')),
                ],
            ],
            [
                'group' => 'admin',
                'priority' => 10,
                'admin' => true,
                'item' => fn (): array => [
                    'label' => 'Platform',
                    'icon' => 'server',
                    'href' => HubRouteNames::url('platform', fallback: '/hub/platform'),
                    'active' => request()->routeIs(HubRouteNames::name('platform'))
                        || request()->routeIs(HubRouteNames::name('platform.user')),
                ],
            ],
        ];
    }

    public function menuPermissions(): array
    {
        return [];
    }

    public function canViewMenu(?object $user, ?object $workspace): bool
    {
        return $user !== null;
    }
}
