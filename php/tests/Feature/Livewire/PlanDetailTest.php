<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\View\Modal\Admin\PlanDetail;
use Core\Tenant\Models\Workspace;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class PlanDetailTest extends LivewireTestCase
{
    private Workspace $workspace;

    private AgentPlan $plan;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
        $this->plan = AgentPlan::factory()->draft()->create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'test-plan',
            'title' => 'Test Plan',
        ]);
    }

    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug]);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->assertOk();
    }

    public function test_mount_loads_plan_by_slug(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug]);

        $this->assertEquals($this->plan->id, $component->instance()->plan->id);
        $this->assertEquals('Test Plan', $component->instance()->plan->title);
    }

    public function test_has_default_modal_states(): void
    {
        $this->actingAsHades();

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->assertSet('showAddTaskModal', false)
            ->assertSet('selectedPhaseId', 0)
            ->assertSet('newTaskName', '')
            ->assertSet('newTaskNotes', '');
    }

    public function test_activate_plan_changes_status(): void
    {
        $this->actingAsHades();

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('activatePlan')
            ->assertDispatched('notify');

        $this->assertEquals(AgentPlan::STATUS_ACTIVE, $this->plan->fresh()->status);
    }

    public function test_complete_plan_changes_status(): void
    {
        $this->actingAsHades();

        $activePlan = AgentPlan::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'active-plan',
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $activePlan->slug])
            ->call('completePlan')
            ->assertDispatched('notify');

        $this->assertEquals(AgentPlan::STATUS_COMPLETED, $activePlan->fresh()->status);
    }

    public function test_archive_plan_changes_status(): void
    {
        $this->actingAsHades();

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('archivePlan')
            ->assertDispatched('notify');

        $this->assertEquals(AgentPlan::STATUS_ARCHIVED, $this->plan->fresh()->status);
    }

    public function test_complete_phase_updates_status(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('completePhase', $phase->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPhase::STATUS_COMPLETED, $phase->fresh()->status);
    }

    public function test_block_phase_updates_status(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('blockPhase', $phase->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPhase::STATUS_BLOCKED, $phase->fresh()->status);
    }

    public function test_skip_phase_updates_status(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('skipPhase', $phase->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPhase::STATUS_SKIPPED, $phase->fresh()->status);
    }

    public function test_reset_phase_restores_to_pending(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->completed()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('resetPhase', $phase->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentPhase::STATUS_PENDING, $phase->fresh()->status);
    }

    public function test_open_add_task_modal_sets_phase_and_shows_modal(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('openAddTaskModal', $phase->id)
            ->assertSet('showAddTaskModal', true)
            ->assertSet('selectedPhaseId', $phase->id)
            ->assertSet('newTaskName', '')
            ->assertSet('newTaskNotes', '');
    }

    public function test_add_task_requires_task_name(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('openAddTaskModal', $phase->id)
            ->set('newTaskName', '')
            ->call('addTask')
            ->assertHasErrors(['newTaskName' => 'required']);
    }

    public function test_add_task_validates_name_max_length(): void
    {
        $this->actingAsHades();

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug])
            ->call('openAddTaskModal', $phase->id)
            ->set('newTaskName', str_repeat('x', 256))
            ->call('addTask')
            ->assertHasErrors(['newTaskName' => 'max']);
    }

    public function test_get_status_color_class_returns_correct_class(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(PlanDetail::class, ['slug' => $this->plan->slug]);
        $instance = $component->instance();

        $this->assertStringContainsString('blue', $instance->getStatusColorClass(AgentPlan::STATUS_ACTIVE));
        $this->assertStringContainsString('green', $instance->getStatusColorClass(AgentPlan::STATUS_COMPLETED));
        $this->assertStringContainsString('red', $instance->getStatusColorClass(AgentPhase::STATUS_BLOCKED));
    }
}
