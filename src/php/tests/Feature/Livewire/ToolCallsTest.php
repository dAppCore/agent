<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\ToolCalls;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

/**
 * Tests for the ToolCalls Livewire component.
 *
 * Note: This component queries McpToolCall from host-uk/core.
 * Tests focus on component state, filters, and actions that do not
 * depend on the mcp_tool_calls table being present.
 */
class ToolCallsTest extends LivewireTestCase
{
    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(ToolCalls::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->assertSet('search', '')
            ->assertSet('server', '')
            ->assertSet('tool', '')
            ->assertSet('status', '')
            ->assertSet('workspace', '')
            ->assertSet('agentType', '')
            ->assertSet('perPage', 25)
            ->assertSet('selectedCallId', null);
    }

    public function test_search_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->set('search', 'plan_create')
            ->assertSet('search', 'plan_create');
    }

    public function test_server_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->set('server', 'agent-server')
            ->assertSet('server', 'agent-server');
    }

    public function test_status_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->set('status', 'success')
            ->assertSet('status', 'success');
    }

    public function test_view_call_sets_selected_call_id(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->call('viewCall', 42)
            ->assertSet('selectedCallId', 42);
    }

    public function test_close_call_detail_clears_selection(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->call('viewCall', 42)
            ->call('closeCallDetail')
            ->assertSet('selectedCallId', null);
    }

    public function test_clear_filters_resets_all(): void
    {
        $this->actingAsHades();

        Livewire::test(ToolCalls::class)
            ->set('search', 'test')
            ->set('server', 'server-1')
            ->set('tool', 'plan_create')
            ->set('status', 'success')
            ->set('workspace', '1')
            ->set('agentType', 'opus')
            ->call('clearFilters')
            ->assertSet('search', '')
            ->assertSet('server', '')
            ->assertSet('tool', '')
            ->assertSet('status', '')
            ->assertSet('workspace', '')
            ->assertSet('agentType', '');
    }

    public function test_get_status_badge_class_returns_green_for_success(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ToolCalls::class);

        $class = $component->instance()->getStatusBadgeClass(true);

        $this->assertStringContainsString('green', $class);
    }

    public function test_get_status_badge_class_returns_red_for_failure(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ToolCalls::class);

        $class = $component->instance()->getStatusBadgeClass(false);

        $this->assertStringContainsString('red', $class);
    }

    public function test_get_agent_badge_class_returns_string(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ToolCalls::class);

        $this->assertNotEmpty($component->instance()->getAgentBadgeClass('opus'));
        $this->assertNotEmpty($component->instance()->getAgentBadgeClass('sonnet'));
        $this->assertNotEmpty($component->instance()->getAgentBadgeClass('unknown'));
    }
}
