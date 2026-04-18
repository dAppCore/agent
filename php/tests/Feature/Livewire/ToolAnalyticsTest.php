<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\ToolAnalytics;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class ToolAnalyticsTest extends LivewireTestCase
{
    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(ToolAnalytics::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->assertSet('days', 7)
            ->assertSet('workspace', '')
            ->assertSet('server', '');
    }

    public function test_set_days_updates_days_property(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->call('setDays', 30)
            ->assertSet('days', 30);
    }

    public function test_set_days_to_seven(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->call('setDays', 30)
            ->call('setDays', 7)
            ->assertSet('days', 7);
    }

    public function test_workspace_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->set('workspace', '1')
            ->assertSet('workspace', '1');
    }

    public function test_server_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->set('server', 'agent-server')
            ->assertSet('server', 'agent-server');
    }

    public function test_clear_filters_resets_all(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolAnalytics::class)
            ->set('workspace', '1')
            ->set('server', 'agent-server')
            ->call('clearFilters')
            ->assertSet('workspace', '')
            ->assertSet('server', '');
    }

    public function test_get_success_rate_color_class_green_above_95(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ToolAnalytics::class);

        $class = $component->instance()->getSuccessRateColorClass(96.0);

        $this->assertStringContainsString('green', $class);
    }

    public function test_get_success_rate_color_class_amber_between_80_and_95(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ToolAnalytics::class);

        $class = $component->instance()->getSuccessRateColorClass(85.0);

        $this->assertStringContainsString('amber', $class);
    }

    public function test_get_success_rate_color_class_red_below_80(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ToolAnalytics::class);

        $class = $component->instance()->getSuccessRateColorClass(70.0);

        $this->assertStringContainsString('red', $class);
    }
}
