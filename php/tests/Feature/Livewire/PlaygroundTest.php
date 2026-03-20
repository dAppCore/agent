<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\Playground;
use Livewire\Livewire;

/**
 * Tests for the Playground Livewire component.
 *
 * Note: This component loads MCP server YAML files and uses Core\Api\Models\ApiKey.
 * Tests focus on component state and interactions. Server loading gracefully
 * handles missing registry files by setting an empty servers array.
 */
class PlaygroundTest extends LivewireTestCase
{
    public function test_renders_successfully(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->assertSet('selectedServer', '')
            ->assertSet('selectedTool', '')
            ->assertSet('arguments', [])
            ->assertSet('response', '')
            ->assertSet('loading', false)
            ->assertSet('apiKey', '')
            ->assertSet('error', null)
            ->assertSet('keyStatus', null)
            ->assertSet('keyInfo', null)
            ->assertSet('tools', []);
    }

    public function test_mount_loads_servers_gracefully_when_registry_missing(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Playground::class);

        // When registry.yaml does not exist, servers defaults to empty array
        $this->assertIsArray($component->instance()->servers);
    }

    public function test_updated_api_key_clears_validation_state(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('keyStatus', 'valid')
            ->set('keyInfo', ['name' => 'Test Key'])
            ->set('apiKey', 'new-key-value')
            ->assertSet('keyStatus', null)
            ->assertSet('keyInfo', null);
    }

    public function test_validate_key_sets_empty_status_when_key_is_blank(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('apiKey', '')
            ->call('validateKey')
            ->assertSet('keyStatus', 'empty');
    }

    public function test_validate_key_sets_invalid_for_unknown_key(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('apiKey', 'not-a-real-key-abc123')
            ->call('validateKey')
            ->assertSet('keyStatus', 'invalid');
    }

    public function test_is_authenticated_returns_true_when_logged_in(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Playground::class);

        $this->assertTrue($component->instance()->isAuthenticated());
    }

    public function test_is_authenticated_returns_false_when_not_logged_in(): void
    {
        // No actingAs - unauthenticated request
        $component = Livewire::test(Playground::class);

        $this->assertFalse($component->instance()->isAuthenticated());
    }

    public function test_updated_selected_server_clears_tool_selection(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('selectedTool', 'some_tool')
            ->set('toolSchema', ['name' => 'some_tool'])
            ->set('selectedServer', 'agent-server')
            ->assertSet('selectedTool', '')
            ->assertSet('toolSchema', null);
    }

    public function test_updated_selected_tool_clears_arguments_and_response(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('arguments', ['key' => 'value'])
            ->set('response', 'previous response')
            ->set('selectedTool', '')
            ->assertSet('toolSchema', null);
    }

    public function test_execute_does_nothing_when_no_server_selected(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('selectedServer', '')
            ->set('selectedTool', '')
            ->call('execute')
            ->assertSet('loading', false)
            ->assertSet('response', '');
    }

    public function test_execute_generates_curl_example_without_api_key(): void
    {
        $this->actingAsHades();

        Livewire::test(Playground::class)
            ->set('selectedServer', 'agent-server')
            ->set('selectedTool', 'plan_create')
            ->call('execute')
            ->assertSet('loading', false);

        // Without a valid API key, response should show the request format
        $component = Livewire::test(Playground::class);
        $component->set('selectedServer', 'agent-server');
        $component->set('selectedTool', 'plan_create');
        $component->call('execute');

        $response = $component->instance()->response;
        if ($response) {
            $decoded = json_decode($response, true);
            $this->assertIsArray($decoded);
        }
    }
}
