<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Menu;

use Core\Mod\Agentic\Mod\Admin\Menu\Contracts\AdminMenuProvider;

class AdminMenuRegistry
{
    /**
     * @var array<int, AdminMenuProvider>
     */
    protected array $providers = [];

    /**
     * @var array<string, array<string, mixed>>
     */
    protected array $groups = [
        'dashboard' => ['label' => 'Dashboard', 'standalone' => true],
        'workspaces' => ['label' => 'Workspaces', 'icon' => 'folders'],
        'services' => ['label' => 'Services', 'standalone' => true],
        'settings' => ['label' => 'Account', 'icon' => 'gear'],
        'admin' => ['label' => 'Admin', 'icon' => 'shield'],
    ];

    public function register(AdminMenuProvider $provider): void
    {
        $this->providers[] = $provider;
    }

    /**
     * @return array<int, AdminMenuProvider>
     */
    public function providers(): array
    {
        return $this->providers;
    }

    /**
     * @return array<string, array{meta: array<string, mixed>, items: array<int, array<string, mixed>>}>
     */
    public function items(?object $user = null, ?object $workspace = null): array
    {
        $grouped = [];
        $isAdmin = method_exists($user, 'isHades') ? (bool) $user->isHades() : false;

        foreach ($this->providers as $provider) {
            if (! $provider->canViewMenu($user, $workspace)) {
                continue;
            }

            foreach ($provider->adminMenuItems() as $registration) {
                if (($registration['admin'] ?? false) && ! $isAdmin) {
                    continue;
                }

                $group = (string) ($registration['group'] ?? 'services');
                $grouped[$group][] = [
                    'priority' => (int) ($registration['priority'] ?? 50),
                    'item' => $registration['item'],
                ];
            }
        }

        $resolved = [];

        foreach ($grouped as $group => $items) {
            usort($items, static fn (array $left, array $right): int => $left['priority'] <=> $right['priority']);

            $resolved[$group] = [
                'meta' => $this->groups[$group] ?? ['label' => ucfirst($group)],
                'items' => array_map(static fn (array $entry): array => $entry['item'](), $items),
            ];
        }

        return $resolved;
    }

    /**
     * Compatibility wrapper for render paths that expect build().
     *
     * @return array<string, array{meta: array<string, mixed>, items: array<int, array<string, mixed>>}>
     */
    public function build(?object $workspace = null, bool $isAdmin = false, ?object $user = null): array
    {
        if ($user !== null && ! $isAdmin && method_exists($user, 'isHades')) {
            $isAdmin = (bool) $user->isHades();
        }

        return $this->items($user, $workspace);
    }
}
