<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mod\Admin;

use Core\Mod\Agentic\Mod\Admin\Menu\AdminMenuRegistry;
use Core\Mod\Agentic\Mod\Admin\Menu\Contracts\AdminMenuProvider;
use Tests\Fixtures\HadesUser;

class AdminMenuRegistryTest extends AdminTestCase
{
    public function test_AdminMenuRegistry_items_Good_groups_and_orders_menu_items(): void
    {
        $registry = new AdminMenuRegistry;
        $registry->register($this->makeProvider());

        $items = $registry->items($this->hadesUser);

        $this->assertArrayHasKey('dashboard', $items);
        $this->assertArrayHasKey('admin', $items);
        $this->assertSame('Dashboard', $items['dashboard']['items'][0]['label']);
        $this->assertSame('Platform', $items['admin']['items'][0]['label']);
    }

    public function test_AdminMenuRegistry_items_Bad_hides_admin_items_for_non_hades_users(): void
    {
        $registry = new AdminMenuRegistry;
        $registry->register($this->makeProvider());

        $user = new class extends HadesUser
        {
            public function isHades(): bool
            {
                return false;
            }
        };

        $items = $registry->items($user);

        $this->assertArrayHasKey('dashboard', $items);
        $this->assertArrayNotHasKey('admin', $items);
    }

    protected function makeProvider(): AdminMenuProvider
    {
        return new class implements AdminMenuProvider
        {
            public function adminMenuItems(): array
            {
                return [
                    [
                        'group' => 'dashboard',
                        'priority' => 10,
                        'item' => fn (): array => ['label' => 'Dashboard', 'href' => '/hub'],
                    ],
                    [
                        'group' => 'admin',
                        'priority' => 20,
                        'admin' => true,
                        'item' => fn (): array => ['label' => 'Platform', 'href' => '/hub/platform'],
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
        };
    }
}
