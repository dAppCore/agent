<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AgentPhaseTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    private AgentPlan $plan;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
        $this->plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);
    }

    public function test_it_can_be_created_with_factory(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $this->assertDatabaseHas('agent_phases', [
            'id' => $phase->id,
            'agent_plan_id' => $this->plan->id,
        ]);
    }

    public function test_it_belongs_to_plan(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $this->assertEquals($this->plan->id, $phase->plan->id);
    }

    public function test_status_helper_methods(): void
    {
        $pending = AgentPhase::factory()->pending()->create(['agent_plan_id' => $this->plan->id]);
        $inProgress = AgentPhase::factory()->inProgress()->create(['agent_plan_id' => $this->plan->id]);
        $completed = AgentPhase::factory()->completed()->create(['agent_plan_id' => $this->plan->id]);
        $blocked = AgentPhase::factory()->blocked()->create(['agent_plan_id' => $this->plan->id]);
        $skipped = AgentPhase::factory()->skipped()->create(['agent_plan_id' => $this->plan->id]);

        $this->assertTrue($pending->isPending());
        $this->assertTrue($inProgress->isInProgress());
        $this->assertTrue($completed->isCompleted());
        $this->assertTrue($blocked->isBlocked());
        $this->assertTrue($skipped->isSkipped());
    }

    public function test_it_can_be_started(): void
    {
        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        $phase->start();

        $this->assertTrue($phase->fresh()->isInProgress());
        $this->assertNotNull($phase->fresh()->started_at);
        $this->assertEquals('1', $this->plan->fresh()->current_phase);
    }

    public function test_it_can_be_completed(): void
    {
        $phase = AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $phase->complete();

        $this->assertTrue($phase->fresh()->isCompleted());
        $this->assertNotNull($phase->fresh()->completed_at);
    }

    public function test_completing_last_phase_completes_plan(): void
    {
        $plan = AgentPlan::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);
        $phase = AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $plan->id,
        ]);

        $phase->complete();

        $this->assertEquals(AgentPlan::STATUS_COMPLETED, $plan->fresh()->status);
    }

    public function test_it_can_be_blocked_with_reason(): void
    {
        $phase = AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $phase->block('Waiting for input');

        $fresh = $phase->fresh();
        $this->assertTrue($fresh->isBlocked());
        $this->assertEquals('Waiting for input', $fresh->metadata['block_reason']);
    }

    public function test_it_can_be_skipped_with_reason(): void
    {
        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $phase->skip('Not applicable');

        $fresh = $phase->fresh();
        $this->assertTrue($fresh->isSkipped());
        $this->assertEquals('Not applicable', $fresh->metadata['skip_reason']);
    }

    public function test_it_can_be_reset(): void
    {
        $phase = AgentPhase::factory()->completed()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $phase->reset();

        $fresh = $phase->fresh();
        $this->assertTrue($fresh->isPending());
        $this->assertNull($fresh->started_at);
        $this->assertNull($fresh->completed_at);
    }

    public function test_it_can_add_task(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
            'tasks' => [],
        ]);

        $phase->addTask('New task', 'Some notes');

        $tasks = $phase->fresh()->getTasks();
        $this->assertCount(1, $tasks);
        $this->assertEquals('New task', $tasks[0]['name']);
        $this->assertEquals('pending', $tasks[0]['status']);
        $this->assertEquals('Some notes', $tasks[0]['notes']);
    }

    public function test_it_can_complete_task_by_index(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
            'tasks' => [
                ['name' => 'Task 1', 'status' => 'pending'],
                ['name' => 'Task 2', 'status' => 'pending'],
            ],
        ]);

        $phase->completeTask(0);

        $tasks = $phase->fresh()->getTasks();
        $this->assertEquals('completed', $tasks[0]['status']);
        $this->assertEquals('pending', $tasks[1]['status']);
    }

    public function test_it_can_complete_task_by_name(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
            'tasks' => [
                ['name' => 'Task 1', 'status' => 'pending'],
                ['name' => 'Task 2', 'status' => 'pending'],
            ],
        ]);

        $phase->completeTask('Task 2');

        $tasks = $phase->fresh()->getTasks();
        $this->assertEquals('pending', $tasks[0]['status']);
        $this->assertEquals('completed', $tasks[1]['status']);
    }

    public function test_it_calculates_task_progress(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
            'tasks' => [
                ['name' => 'Task 1', 'status' => 'completed'],
                ['name' => 'Task 2', 'status' => 'pending'],
                ['name' => 'Task 3', 'status' => 'pending'],
                ['name' => 'Task 4', 'status' => 'completed'],
            ],
        ]);

        $progress = $phase->getTaskProgress();

        $this->assertEquals(4, $progress['total']);
        $this->assertEquals(2, $progress['completed']);
        $this->assertEquals(2, $progress['remaining']);
        $this->assertEquals(50, $progress['percentage']);
    }

    public function test_it_gets_remaining_tasks(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
            'tasks' => [
                ['name' => 'Task 1', 'status' => 'completed'],
                ['name' => 'Task 2', 'status' => 'pending'],
                ['name' => 'Task 3', 'status' => 'pending'],
            ],
        ]);

        $remaining = $phase->getRemainingTasks();

        $this->assertCount(2, $remaining);
        $this->assertContains('Task 2', $remaining);
        $this->assertContains('Task 3', $remaining);
    }

    public function test_all_tasks_complete_returns_correctly(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
            'tasks' => [
                ['name' => 'Task 1', 'status' => 'completed'],
                ['name' => 'Task 2', 'status' => 'completed'],
            ],
        ]);

        $this->assertTrue($phase->allTasksComplete());

        $phase->addTask('New task');

        $this->assertFalse($phase->fresh()->allTasksComplete());
    }

    public function test_it_can_add_checkpoint(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $phase->addCheckpoint('Reached midpoint', ['progress' => 50]);

        $checkpoints = $phase->fresh()->getCheckpoints();
        $this->assertCount(1, $checkpoints);
        $this->assertEquals('Reached midpoint', $checkpoints[0]['note']);
        $this->assertEquals(['progress' => 50], $checkpoints[0]['context']);
        $this->assertNotNull($checkpoints[0]['timestamp']);
    }

    public function test_dependency_checking(): void
    {
        $dep1 = AgentPhase::factory()->completed()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);
        $dep2 = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 2,
        ]);

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 3,
            'dependencies' => [$dep1->id, $dep2->id],
        ]);

        $blockers = $phase->checkDependencies();

        $this->assertCount(1, $blockers);
        $this->assertEquals($dep2->id, $blockers[0]['phase_id']);
    }

    public function test_check_dependencies_returns_empty_when_no_dependencies(): void
    {
        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'dependencies' => null,
        ]);

        $this->assertSame([], $phase->checkDependencies());
    }

    public function test_check_dependencies_not_blocked_by_skipped_phase(): void
    {
        $dep = AgentPhase::factory()->skipped()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 2,
            'dependencies' => [$dep->id],
        ]);

        $this->assertSame([], $phase->checkDependencies());
        $this->assertTrue($phase->canStart());
    }

    public function test_check_dependencies_uses_single_query_for_multiple_deps(): void
    {
        $deps = AgentPhase::factory()->pending()->count(5)->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'dependencies' => $deps->pluck('id')->toArray(),
        ]);

        $queryCount = 0;
        \DB::listen(function () use (&$queryCount) {
            $queryCount++;
        });

        $blockers = $phase->checkDependencies();

        $this->assertCount(5, $blockers);
        $this->assertSame(1, $queryCount, 'checkDependencies() should issue exactly one query');
    }

    public function test_check_dependencies_blocker_contains_expected_keys(): void
    {
        $dep = AgentPhase::factory()->inProgress()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
            'name' => 'Blocker Phase',
        ]);

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 2,
            'dependencies' => [$dep->id],
        ]);

        $blockers = $phase->checkDependencies();

        $this->assertCount(1, $blockers);
        $this->assertEquals($dep->id, $blockers[0]['phase_id']);
        $this->assertEquals(1, $blockers[0]['phase_order']);
        $this->assertEquals('Blocker Phase', $blockers[0]['phase_name']);
        $this->assertEquals(AgentPhase::STATUS_IN_PROGRESS, $blockers[0]['status']);
    }

    public function test_can_start_checks_dependencies(): void
    {
        $dep = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 1,
        ]);

        $phase = AgentPhase::factory()->pending()->create([
            'agent_plan_id' => $this->plan->id,
            'order' => 2,
            'dependencies' => [$dep->id],
        ]);

        $this->assertFalse($phase->canStart());

        $dep->update(['status' => AgentPhase::STATUS_COMPLETED]);

        $this->assertTrue($phase->fresh()->canStart());
    }

    public function test_status_icons(): void
    {
        $pending = AgentPhase::factory()->pending()->make();
        $inProgress = AgentPhase::factory()->inProgress()->make();
        $completed = AgentPhase::factory()->completed()->make();
        $blocked = AgentPhase::factory()->blocked()->make();
        $skipped = AgentPhase::factory()->skipped()->make();

        $this->assertEquals('⬜', $pending->getStatusIcon());
        $this->assertEquals('🔄', $inProgress->getStatusIcon());
        $this->assertEquals('✅', $completed->getStatusIcon());
        $this->assertEquals('🚫', $blocked->getStatusIcon());
        $this->assertEquals('⏭️', $skipped->getStatusIcon());
    }

    public function test_to_mcp_context_returns_array(): void
    {
        $phase = AgentPhase::factory()->create([
            'agent_plan_id' => $this->plan->id,
        ]);

        $context = $phase->toMcpContext();

        $this->assertIsArray($context);
        $this->assertArrayHasKey('id', $context);
        $this->assertArrayHasKey('order', $context);
        $this->assertArrayHasKey('name', $context);
        $this->assertArrayHasKey('status', $context);
        $this->assertArrayHasKey('task_progress', $context);
        $this->assertArrayHasKey('can_start', $context);
    }

    public function test_scopes_work_correctly(): void
    {
        AgentPhase::factory()->pending()->create(['agent_plan_id' => $this->plan->id]);
        AgentPhase::factory()->inProgress()->create(['agent_plan_id' => $this->plan->id]);
        AgentPhase::factory()->completed()->create(['agent_plan_id' => $this->plan->id]);
        AgentPhase::factory()->blocked()->create(['agent_plan_id' => $this->plan->id]);

        $this->assertCount(1, AgentPhase::pending()->get());
        $this->assertCount(1, AgentPhase::inProgress()->get());
        $this->assertCount(1, AgentPhase::completed()->get());
        $this->assertCount(1, AgentPhase::blocked()->get());
    }
}
