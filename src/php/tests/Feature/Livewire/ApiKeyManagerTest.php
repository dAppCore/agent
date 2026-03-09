<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\ApiKeyManager;
use Core\Tenant\Models\Workspace;
use Livewire\Livewire;

/**
 * Tests for the ApiKeyManager Livewire component.
 *
 * Note: This component manages workspace API keys via Core\Api\Models\ApiKey
 * (from host-uk/core). Tests for key creation require the full core package
 * to be installed. Tests here focus on component state and validation.
 */
class ApiKeyManagerTest extends LivewireTestCase
{
    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    public function test_renders_successfully_with_workspace(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->assertOk();
    }

    public function test_mount_loads_workspace(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace]);

        $this->assertEquals($this->workspace->id, $component->instance()->workspace->id);
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->assertSet('showCreateModal', false)
            ->assertSet('newKeyName', '')
            ->assertSet('newKeyExpiry', 'never')
            ->assertSet('showNewKeyModal', false)
            ->assertSet('newPlainKey', null);
    }

    public function test_open_create_modal_shows_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->call('openCreateModal')
            ->assertSet('showCreateModal', true);
    }

    public function test_open_create_modal_resets_form(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->set('newKeyName', 'Old Name')
            ->call('openCreateModal')
            ->assertSet('newKeyName', '')
            ->assertSet('newKeyExpiry', 'never');
    }

    public function test_close_create_modal_hides_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->call('openCreateModal')
            ->call('closeCreateModal')
            ->assertSet('showCreateModal', false);
    }

    public function test_create_key_requires_name(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->call('openCreateModal')
            ->set('newKeyName', '')
            ->call('createKey')
            ->assertHasErrors(['newKeyName' => 'required']);
    }

    public function test_create_key_validates_name_max_length(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->call('openCreateModal')
            ->set('newKeyName', str_repeat('x', 101))
            ->call('createKey')
            ->assertHasErrors(['newKeyName' => 'max']);
    }

    public function test_toggle_scope_adds_scope_if_not_present(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->set('newKeyScopes', [])
            ->call('toggleScope', 'read')
            ->assertSet('newKeyScopes', ['read']);
    }

    public function test_toggle_scope_removes_scope_if_already_present(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->set('newKeyScopes', ['read', 'write'])
            ->call('toggleScope', 'read')
            ->assertSet('newKeyScopes', ['write']);
    }

    public function test_close_new_key_modal_clears_plain_key(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeyManager::class, ['workspace' => $this->workspace])
            ->set('newPlainKey', 'secret-key-value')
            ->set('showNewKeyModal', true)
            ->call('closeNewKeyModal')
            ->assertSet('newPlainKey', null)
            ->assertSet('showNewKeyModal', false);
    }
}
