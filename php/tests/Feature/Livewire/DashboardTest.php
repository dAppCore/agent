<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\Dashboard;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class DashboardTest extends LivewireTestCase
{
    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(Dashboard::class);
    }

    public function test_unauthenticated_user_cannot_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(Dashboard::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(Dashboard::class)
            ->assertOk();
    }

    public function test_refresh_dispatches_notify_event(): void
    {
        $this->actingAsHades();

        Livewire::test(Dashboard::class)
            ->call('refresh')
            ->assertDispatched('notify');
    }

    public function test_has_correct_initial_properties(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Dashboard::class);

        $component->assertOk();
    }

    public function test_stats_returns_array_with_expected_keys(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Dashboard::class);

        $stats = $component->instance()->stats;

        $this->assertIsArray($stats);
        $this->assertArrayHasKey('active_plans', $stats);
        $this->assertArrayHasKey('total_plans', $stats);
        $this->assertArrayHasKey('active_sessions', $stats);
        $this->assertArrayHasKey('today_sessions', $stats);
        $this->assertArrayHasKey('tool_calls_7d', $stats);
        $this->assertArrayHasKey('success_rate', $stats);
    }

    public function test_stat_cards_returns_four_items(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Dashboard::class);

        $cards = $component->instance()->statCards;

        $this->assertIsArray($cards);
        $this->assertCount(4, $cards);
    }

    public function test_blocked_alert_is_null_when_no_blocked_plans(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Dashboard::class);

        $this->assertNull($component->instance()->blockedAlert);
    }

    public function test_quick_links_returns_four_items(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Dashboard::class);

        $links = $component->instance()->quickLinks;

        $this->assertIsArray($links);
        $this->assertCount(4, $links);
    }
}
