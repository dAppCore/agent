<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\View\Modal\Admin\ApiKeys;
use Core\Tenant\Models\Workspace;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class ApiKeysTest extends LivewireTestCase
{
    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(ApiKeys::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->assertSet('workspace', '')
            ->assertSet('status', '')
            ->assertSet('perPage', 25)
            ->assertSet('showCreateModal', false)
            ->assertSet('showEditModal', false);
    }

    public function test_open_create_modal_shows_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->call('openCreateModal')
            ->assertSet('showCreateModal', true);
    }

    public function test_close_create_modal_hides_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->call('openCreateModal')
            ->call('closeCreateModal')
            ->assertSet('showCreateModal', false);
    }

    public function test_open_create_modal_resets_form_fields(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->set('newKeyName', 'Old Name')
            ->call('openCreateModal')
            ->assertSet('newKeyName', '')
            ->assertSet('newKeyPermissions', [])
            ->assertSet('newKeyRateLimit', 100);
    }

    public function test_create_key_requires_name(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->call('openCreateModal')
            ->set('newKeyName', '')
            ->set('newKeyWorkspace', $this->workspace->id)
            ->set('newKeyPermissions', [AgentApiKey::PERM_PLANS_READ])
            ->call('createKey')
            ->assertHasErrors(['newKeyName' => 'required']);
    }

    public function test_create_key_requires_at_least_one_permission(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->call('openCreateModal')
            ->set('newKeyName', 'Test Key')
            ->set('newKeyWorkspace', $this->workspace->id)
            ->set('newKeyPermissions', [])
            ->call('createKey')
            ->assertHasErrors(['newKeyPermissions']);
    }

    public function test_create_key_requires_valid_workspace(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->call('openCreateModal')
            ->set('newKeyName', 'Test Key')
            ->set('newKeyWorkspace', 99999)
            ->set('newKeyPermissions', [AgentApiKey::PERM_PLANS_READ])
            ->call('createKey')
            ->assertHasErrors(['newKeyWorkspace' => 'exists']);
    }

    public function test_create_key_validates_rate_limit_minimum(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->call('openCreateModal')
            ->set('newKeyName', 'Test Key')
            ->set('newKeyWorkspace', $this->workspace->id)
            ->set('newKeyPermissions', [AgentApiKey::PERM_PLANS_READ])
            ->set('newKeyRateLimit', 0)
            ->call('createKey')
            ->assertHasErrors(['newKeyRateLimit' => 'min']);
    }

    public function test_revoke_key_marks_key_as_revoked(): void
    {
        $this->actingAsHades();

        $key = AgentApiKey::generate($this->workspace, 'Test Key', [AgentApiKey::PERM_PLANS_READ]);

        Livewire::test(ApiKeys::class)
            ->call('revokeKey', $key->id)
            ->assertOk();

        $this->assertNotNull($key->fresh()->revoked_at);
    }

    public function test_clear_filters_resets_workspace_and_status(): void
    {
        $this->actingAsHades();

        Livewire::test(ApiKeys::class)
            ->set('workspace', '1')
            ->set('status', 'active')
            ->call('clearFilters')
            ->assertSet('workspace', '')
            ->assertSet('status', '');
    }

    public function test_open_edit_modal_populates_fields(): void
    {
        $this->actingAsHades();

        $key = AgentApiKey::generate(
            $this->workspace,
            'Edit Me',
            [AgentApiKey::PERM_PLANS_READ],
            200
        );

        Livewire::test(ApiKeys::class)
            ->call('openEditModal', $key->id)
            ->assertSet('showEditModal', true)
            ->assertSet('editingKeyId', $key->id)
            ->assertSet('editingRateLimit', 200);
    }

    public function test_close_edit_modal_clears_editing_state(): void
    {
        $this->actingAsHades();

        $key = AgentApiKey::generate($this->workspace, 'Test Key', [AgentApiKey::PERM_PLANS_READ]);

        Livewire::test(ApiKeys::class)
            ->call('openEditModal', $key->id)
            ->call('closeEditModal')
            ->assertSet('showEditModal', false)
            ->assertSet('editingKeyId', null);
    }

    public function test_get_status_badge_class_returns_green_for_active_key(): void
    {
        $this->actingAsHades();

        $key = AgentApiKey::generate($this->workspace, 'Active Key', [AgentApiKey::PERM_PLANS_READ]);

        $component = Livewire::test(ApiKeys::class);
        $class = $component->instance()->getStatusBadgeClass($key->fresh());

        $this->assertStringContainsString('green', $class);
    }

    public function test_get_status_badge_class_returns_red_for_revoked_key(): void
    {
        $this->actingAsHades();

        $key = AgentApiKey::generate($this->workspace, 'Revoked Key', [AgentApiKey::PERM_PLANS_READ]);
        $key->update(['revoked_at' => now()]);

        $component = Livewire::test(ApiKeys::class);
        $class = $component->instance()->getStatusBadgeClass($key->fresh());

        $this->assertStringContainsString('red', $class);
    }

    public function test_stats_returns_array_with_expected_keys(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ApiKeys::class);
        $stats = $component->instance()->stats;

        $this->assertArrayHasKey('total', $stats);
        $this->assertArrayHasKey('active', $stats);
        $this->assertArrayHasKey('revoked', $stats);
        $this->assertArrayHasKey('total_calls', $stats);
    }

    public function test_available_permissions_returns_all_permissions(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(ApiKeys::class);
        $permissions = $component->instance()->availablePermissions;

        $this->assertIsArray($permissions);
        $this->assertNotEmpty($permissions);
    }
}
