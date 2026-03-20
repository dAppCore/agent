<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\View\Modal\Admin\Plans;
use Core\Tenant\Models\Workspace;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class PlansTest extends LivewireTestCase
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

        Livewire::test(Plans::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(Plans::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(Plans::class)
            ->assertSet('search', '')
            ->assertSet('status', '')
            ->assertSet('workspace', '')
            ->assertSet('perPage', 15);
    }

    public function test_search_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Plans::class)
            ->set('search', 'my plan')
            ->assertSet('search', 'my plan');
    }

    public function test_status_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Plans::class)
            ->set('status', AgentPlan::STATUS_ACTIVE)
            ->assertSet('status', AgentPlan::STATUS_ACTIVE);
    }

    public function test_workspace_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Plans::class)
            ->set('workspace', (string) $this->workspace->id)
            ->assertSet('workspace', (string) $this->workspace->id);
    }

    public function test_clear_filters_resets_all_filters(): void
    {
        $this->actingAsHades();

        Livewire::test(Plans::class)
            ->set('search', 'test')
            ->set('status', AgentPlan::STATUS_ACTIVE)
            ->set('workspace', (string) $this->workspace->id)
            ->call('clearFilters')
            ->assertSet('search', '')
            ->assertSet('status', '')
            ->assertSet('workspace', '');
    }

    public function test_activate_plan_changes_status_to_active(): void
    {
        $this->actingAsHades();

        $plan = AgentPlan::factory()->draft()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Plans::class)
            ->call('activate', $plan->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPlan::STATUS_ACTIVE, $plan->fresh()->status);
    }

    public function test_complete_plan_changes_status_to_completed(): void
    {
        $this->actingAsHades();

        $plan = AgentPlan::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Plans::class)
            ->call('complete', $plan->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPlan::STATUS_COMPLETED, $plan->fresh()->status);
    }

    public function test_archive_plan_changes_status_to_archived(): void
    {
        $this->actingAsHades();

        $plan = AgentPlan::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Plans::class)
            ->call('archive', $plan->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPlan::STATUS_ARCHIVED, $plan->fresh()->status);
    }

    public function test_delete_plan_removes_from_database(): void
    {
        $this->actingAsHades();

        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $planId = $plan->id;

        Livewire::test(Plans::class)
            ->call('delete', $planId)
            ->assertDispatched('notify');

        $this->assertDatabaseMissing('agent_plans', ['id' => $planId]);
    }

    public function test_status_options_returns_all_statuses(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Plans::class);

        $options = $component->instance()->statusOptions;

        $this->assertArrayHasKey(AgentPlan::STATUS_DRAFT, $options);
        $this->assertArrayHasKey(AgentPlan::STATUS_ACTIVE, $options);
        $this->assertArrayHasKey(AgentPlan::STATUS_COMPLETED, $options);
        $this->assertArrayHasKey(AgentPlan::STATUS_ARCHIVED, $options);
    }
}
