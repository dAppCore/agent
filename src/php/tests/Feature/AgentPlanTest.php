<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AgentPlanTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    public function test_it_can_be_created_with_factory(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $this->assertDatabaseHas('agent_plans', [
            'id' => $plan->id,
            'workspace_id' => $this->workspace->id,
        ]);
    }

    public function test_it_has_correct_default_status(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $this->assertEquals(AgentPlan::STATUS_DRAFT, $plan->status);
    }

    public function test_it_can_be_activated(): void
    {
        $plan = AgentPlan::factory()->draft()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $plan->activate();

        $this->assertEquals(AgentPlan::STATUS_ACTIVE, $plan->fresh()->status);
    }

    public function test_it_can_be_completed(): void
    {
        $plan = AgentPlan::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $plan->complete();

        $this->assertEquals(AgentPlan::STATUS_COMPLETED, $plan->fresh()->status);
    }

    public function test_it_can_be_archived_with_reason(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $plan->archive('No longer needed');

        $fresh = $plan->fresh();
        $this->assertEquals(AgentPlan::STATUS_ARCHIVED, $fresh->status);
        $this->assertEquals('No longer needed', $fresh->metadata['archive_reason']);
        $this->assertNotNull($fresh->archived_at);
    }

    public function test_it_generates_unique_slugs(): void
    {
        AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'test-plan',
        ]);

        $slug = AgentPlan::generateSlug('Test Plan');

        $this->assertEquals('test-plan-1', $slug);
    }

    public function test_it_calculates_progress_correctly(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        AgentPhase::factory()->count(2)->completed()->create([
            'agent_plan_id' => $plan->id,
        ]);
        AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $plan->id,
        ]);
        AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $plan->id,
        ]);

        $progress = $plan->getProgress();

        $this->assertEquals(4, $progress['total']);
        $this->assertEquals(2, $progress['completed']);
        $this->assertEquals(1, $progress['in_progress']);
        $this->assertEquals(1, $progress['pending']);
        $this->assertEquals(50, $progress['percentage']);
    }

    public function test_it_checks_all_phases_complete(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        AgentPhase::factory()->count(2)->completed()->create([
            'agent_plan_id' => $plan->id,
        ]);
        AgentPhase::factory()->skipped()->create([
            'agent_plan_id' => $plan->id,
        ]);

        $this->assertTrue($plan->checkAllPhasesComplete());

        AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $plan->id,
        ]);

        $this->assertFalse($plan->fresh()->checkAllPhasesComplete());
    }

    public function test_it_gets_current_phase(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'current_phase' => '2',
        ]);

        $phase1 = AgentPhase::factory()->create([
            'agent_plan_id' => $plan->id,
            'order' => 1,
            'name' => 'Phase One',
        ]);
        $phase2 = AgentPhase::factory()->create([
            'agent_plan_id' => $plan->id,
            'order' => 2,
            'name' => 'Phase Two',
        ]);

        $current = $plan->getCurrentPhase();

        $this->assertEquals($phase2->id, $current->id);
    }

    public function test_it_returns_first_phase_when_current_is_null(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'current_phase' => null,
        ]);

        $phase1 = AgentPhase::factory()->create([
            'agent_plan_id' => $plan->id,
            'order' => 1,
        ]);
        AgentPhase::factory()->create([
            'agent_plan_id' => $plan->id,
            'order' => 2,
        ]);

        $current = $plan->getCurrentPhase();

        $this->assertEquals($phase1->id, $current->id);
    }

    public function test_active_scope_works(): void
    {
        AgentPlan::factory()->draft()->create(['workspace_id' => $this->workspace->id]);
        AgentPlan::factory()->active()->create(['workspace_id' => $this->workspace->id]);
        AgentPlan::factory()->completed()->create(['workspace_id' => $this->workspace->id]);

        $active = AgentPlan::active()->get();

        $this->assertCount(1, $active);
        $this->assertEquals(AgentPlan::STATUS_ACTIVE, $active->first()->status);
    }

    public function test_not_archived_scope_works(): void
    {
        AgentPlan::factory()->draft()->create(['workspace_id' => $this->workspace->id]);
        AgentPlan::factory()->active()->create(['workspace_id' => $this->workspace->id]);
        AgentPlan::factory()->archived()->create(['workspace_id' => $this->workspace->id]);

        $notArchived = AgentPlan::notArchived()->get();

        $this->assertCount(2, $notArchived);
    }

    public function test_to_markdown_generates_output(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Test Plan',
            'description' => 'A test description',
        ]);

        AgentPhase::factory()->completed()->create([
            'agent_plan_id' => $plan->id,
            'order' => 1,
            'name' => 'Phase One',
        ]);

        $markdown = $plan->toMarkdown();

        $this->assertStringContainsString('# Test Plan', $markdown);
        $this->assertStringContainsString('A test description', $markdown);
        $this->assertStringContainsString('Phase One', $markdown);
    }

    public function test_to_mcp_context_returns_array(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $context = $plan->toMcpContext();

        $this->assertIsArray($context);
        $this->assertArrayHasKey('slug', $context);
        $this->assertArrayHasKey('title', $context);
        $this->assertArrayHasKey('status', $context);
        $this->assertArrayHasKey('progress', $context);
        $this->assertArrayHasKey('phases', $context);
    }

    public function test_with_phases_factory_state(): void
    {
        $plan = AgentPlan::factory()->withPhases(3)->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $this->assertCount(3, $plan->phases);
        $this->assertEquals(1, $plan->phases[0]['order']);
        $this->assertEquals(2, $plan->phases[1]['order']);
        $this->assertEquals(3, $plan->phases[2]['order']);
    }
}
