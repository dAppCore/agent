<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\RequestLog;
use Livewire\Livewire;

/**
 * Tests for the RequestLog Livewire component.
 *
 * Note: This component queries McpApiRequest from host-uk/core.
 * Tests focus on component state and interactions that do not
 * require the mcp_api_requests table to be present.
 */
class RequestLogTest extends LivewireTestCase
{
    public function test_renders_successfully(): void
    {
        $this->actingAsHades();

        Livewire::test(RequestLog::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(RequestLog::class)
            ->assertSet('serverFilter', '')
            ->assertSet('statusFilter', '')
            ->assertSet('selectedRequestId', null)
            ->assertSet('selectedRequest', null);
    }

    public function test_server_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(RequestLog::class)
            ->set('serverFilter', 'agent-server')
            ->assertSet('serverFilter', 'agent-server');
    }

    public function test_status_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(RequestLog::class)
            ->set('statusFilter', 'success')
            ->assertSet('statusFilter', 'success');
    }

    public function test_close_detail_clears_selection(): void
    {
        $this->actingAsHades();

        Livewire::test(RequestLog::class)
            ->set('selectedRequestId', 5)
            ->call('closeDetail')
            ->assertSet('selectedRequestId', null)
            ->assertSet('selectedRequest', null);
    }

    public function test_updated_server_filter_triggers_re_render(): void
    {
        $this->actingAsHades();

        // Setting filter should update the property (pagination resets internally)
        Livewire::test(RequestLog::class)
            ->set('serverFilter', 'my-server')
            ->assertSet('serverFilter', 'my-server')
            ->assertOk();
    }

    public function test_updated_status_filter_triggers_re_render(): void
    {
        $this->actingAsHades();

        Livewire::test(RequestLog::class)
            ->set('statusFilter', 'failed')
            ->assertSet('statusFilter', 'failed')
            ->assertOk();
    }
}
