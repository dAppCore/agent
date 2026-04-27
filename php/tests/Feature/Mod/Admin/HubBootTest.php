<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mod\Admin;

use Core\Events\AdminPanelBooting;
use Core\Events\DomainResolving;
use Core\Mod\Agentic\Mod\Admin\Menu\AdminMenuRegistry;
use Core\Mod\Agentic\Website\Hub\Boot;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;

class HubBootTest extends AdminTestCase
{
    public function test_HubBoot_onDomainResolving_Good_registers_matching_domains(): void
    {
        $boot = new Boot($this->app);
        $event = new DomainResolving('hub.core.test');

        $boot->onDomainResolving($event);

        $this->assertSame(Boot::class, $event->matchedProvider());
    }

    public function test_HubBoot_onDomainResolving_Bad_ignores_non_matching_domains(): void
    {
        $boot = new Boot($this->app);
        $event = new DomainResolving('example.com');

        $boot->onDomainResolving($event);

        $this->assertNull($event->matchedProvider());
    }

    public function test_HubBoot_onAdminPanel_Good_registers_global_components_and_secondary_routes(): void
    {
        $boot = new Boot($this->app);
        $event = new AdminPanelBooting;

        $this->app->instance('request', Request::create('http://hub.core.test/hub'));
        $boot->onAdminPanel($event);

        $this->assertCount(1, $event->viewRequests());
        $this->assertCount(1, $event->translationRequests());
        $this->assertCount(2, $event->livewireRequests());
        $this->assertCount(1, $event->routeRequests());
        $this->assertCount(1, app(AdminMenuRegistry::class)->providers());

        ($event->routeRequests()[0])();

        $this->assertTrue(Route::has('hub_core_test.dashboard'));
        $this->assertTrue(Route::has('hub_core_test.platform.user'));
    }
}
