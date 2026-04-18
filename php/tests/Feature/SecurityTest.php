<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Mcp\Tools\Agent\Plan\PlanGet;
use Core\Mod\Agentic\Mcp\Tools\Agent\Plan\PlanList;
use Core\Mod\Agentic\Mcp\Tools\Agent\State\StateGet;
use Core\Mod\Agentic\Mcp\Tools\Agent\State\StateList;
use Core\Mod\Agentic\Mcp\Tools\Agent\State\StateSet;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\WorkspaceState;
use Core\Mod\Agentic\Models\Task;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Security tests for workspace isolation and SQL injection prevention.
 */
class SecurityTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    private Workspace $otherWorkspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
        $this->otherWorkspace = Workspace::factory()->create();
    }

    // =========================================================================
    // StateSet Workspace Scoping Tests
    // =========================================================================

    public function test_state_set_requires_workspace_context(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $tool = new StateSet;
        $result = $tool->handle([
            'plan_slug' => $plan->slug,
            'key' => 'test_key',
            'value' => 'test_value',
        ], []); // No workspace_id in context

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('workspace_id is required', $result['error']);
    }

    public function test_state_set_cannot_access_other_workspace_plans(): void
    {
        $otherPlan = AgentPlan::factory()->create([
            'workspace_id' => $this->otherWorkspace->id,
        ]);

        $tool = new StateSet;
        $result = $tool->handle([
            'plan_slug' => $otherPlan->slug,
            'key' => 'test_key',
            'value' => 'test_value',
        ], ['workspace_id' => $this->workspace->id]); // Different workspace

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('Plan not found', $result['error']);
    }

    public function test_state_set_works_with_correct_workspace(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $tool = new StateSet;
        $result = $tool->handle([
            'plan_slug' => $plan->slug,
            'key' => 'test_key',
            'value' => 'test_value',
        ], ['workspace_id' => $this->workspace->id]);

        $this->assertArrayHasKey('success', $result);
        $this->assertTrue($result['success']);
        $this->assertEquals('test_key', $result['state']['key']);
    }

    // =========================================================================
    // StateGet Workspace Scoping Tests
    // =========================================================================

    public function test_state_get_requires_workspace_context(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        WorkspaceState::create([
            'agent_plan_id' => $plan->id,
            'key' => 'test_key',
            'value' => ['data' => 'secret'],
        ]);

        $tool = new StateGet;
        $result = $tool->handle([
            'plan_slug' => $plan->slug,
            'key' => 'test_key',
        ], []); // No workspace_id in context

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('workspace_id is required', $result['error']);
    }

    public function test_state_get_cannot_access_other_workspace_state(): void
    {
        $otherPlan = AgentPlan::factory()->create([
            'workspace_id' => $this->otherWorkspace->id,
        ]);

        WorkspaceState::create([
            'agent_plan_id' => $otherPlan->id,
            'key' => 'secret_key',
            'value' => ['data' => 'sensitive'],
        ]);

        $tool = new StateGet;
        $result = $tool->handle([
            'plan_slug' => $otherPlan->slug,
            'key' => 'secret_key',
        ], ['workspace_id' => $this->workspace->id]); // Different workspace

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('Plan not found', $result['error']);
    }

    public function test_state_get_works_with_correct_workspace(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        WorkspaceState::create([
            'agent_plan_id' => $plan->id,
            'key' => 'test_key',
            'value' => ['data' => 'allowed'],
        ]);

        $tool = new StateGet;
        $result = $tool->handle([
            'plan_slug' => $plan->slug,
            'key' => 'test_key',
        ], ['workspace_id' => $this->workspace->id]);

        $this->assertArrayHasKey('success', $result);
        $this->assertTrue($result['success']);
        $this->assertEquals('test_key', $result['key']);
    }

    // =========================================================================
    // StateList Workspace Scoping Tests
    // =========================================================================

    public function test_state_list_requires_workspace_context(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $tool = new StateList;
        $result = $tool->handle([
            'plan_slug' => $plan->slug,
        ], []); // No workspace_id in context

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('workspace_id is required', $result['error']);
    }

    public function test_state_list_cannot_access_other_workspace_states(): void
    {
        $otherPlan = AgentPlan::factory()->create([
            'workspace_id' => $this->otherWorkspace->id,
        ]);

        WorkspaceState::create([
            'agent_plan_id' => $otherPlan->id,
            'key' => 'secret_key',
            'value' => ['data' => 'sensitive'],
        ]);

        $tool = new StateList;
        $result = $tool->handle([
            'plan_slug' => $otherPlan->slug,
        ], ['workspace_id' => $this->workspace->id]); // Different workspace

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('Plan not found', $result['error']);
    }

    // =========================================================================
    // PlanGet Workspace Scoping Tests
    // =========================================================================

    public function test_plan_get_requires_workspace_context(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $tool = new PlanGet;
        $result = $tool->handle([
            'slug' => $plan->slug,
        ], []); // No workspace_id in context

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('workspace_id is required', $result['error']);
    }

    public function test_plan_get_cannot_access_other_workspace_plans(): void
    {
        $otherPlan = AgentPlan::factory()->create([
            'workspace_id' => $this->otherWorkspace->id,
            'title' => 'Secret Plan',
        ]);

        $tool = new PlanGet;
        $result = $tool->handle([
            'slug' => $otherPlan->slug,
        ], ['workspace_id' => $this->workspace->id]); // Different workspace

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('Plan not found', $result['error']);
    }

    public function test_plan_get_works_with_correct_workspace(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'My Plan',
        ]);

        $tool = new PlanGet;
        $result = $tool->handle([
            'slug' => $plan->slug,
        ], ['workspace_id' => $this->workspace->id]);

        $this->assertArrayHasKey('success', $result);
        $this->assertTrue($result['success']);
        $this->assertEquals('My Plan', $result['plan']['title']);
    }

    // =========================================================================
    // PlanList Workspace Scoping Tests
    // =========================================================================

    public function test_plan_list_requires_workspace_context(): void
    {
        $tool = new PlanList;
        $result = $tool->handle([], []); // No workspace_id in context

        $this->assertArrayHasKey('error', $result);
        $this->assertStringContainsString('workspace_id is required', $result['error']);
    }

    public function test_plan_list_only_returns_workspace_plans(): void
    {
        // Create plans in both workspaces
        AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'My Plan',
        ]);
        AgentPlan::factory()->create([
            'workspace_id' => $this->otherWorkspace->id,
            'title' => 'Other Plan',
        ]);

        $tool = new PlanList;
        $result = $tool->handle([], ['workspace_id' => $this->workspace->id]);

        $this->assertArrayHasKey('success', $result);
        $this->assertTrue($result['success']);
        $this->assertEquals(1, $result['total']);
        $this->assertEquals('My Plan', $result['plans'][0]['title']);
    }

    // =========================================================================
    // Task Model Ordering Tests (SQL Injection Prevention)
    // =========================================================================

    public function test_task_order_by_priority_uses_parameterised_query(): void
    {
        Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Low task',
            'priority' => 'low',
            'status' => 'pending',
        ]);
        Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Urgent task',
            'priority' => 'urgent',
            'status' => 'pending',
        ]);
        Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'High task',
            'priority' => 'high',
            'status' => 'pending',
        ]);

        $tasks = Task::forWorkspace($this->workspace->id)
            ->orderByPriority()
            ->get();

        $this->assertEquals('Urgent task', $tasks[0]->title);
        $this->assertEquals('High task', $tasks[1]->title);
        $this->assertEquals('Low task', $tasks[2]->title);
    }

    public function test_task_order_by_status_uses_parameterised_query(): void
    {
        Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Done task',
            'priority' => 'normal',
            'status' => 'done',
        ]);
        Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'In progress task',
            'priority' => 'normal',
            'status' => 'in_progress',
        ]);
        Task::create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Pending task',
            'priority' => 'normal',
            'status' => 'pending',
        ]);

        $tasks = Task::forWorkspace($this->workspace->id)
            ->orderByStatus()
            ->get();

        $this->assertEquals('In progress task', $tasks[0]->title);
        $this->assertEquals('Pending task', $tasks[1]->title);
        $this->assertEquals('Done task', $tasks[2]->title);
    }

    // =========================================================================
    // AgentPlan Model Ordering Tests (SQL Injection Prevention)
    // =========================================================================

    public function test_plan_order_by_status_uses_parameterised_query(): void
    {
        AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Archived plan',
            'status' => AgentPlan::STATUS_ARCHIVED,
        ]);
        AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Active plan',
            'status' => AgentPlan::STATUS_ACTIVE,
        ]);
        AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Draft plan',
            'status' => AgentPlan::STATUS_DRAFT,
        ]);

        $plans = AgentPlan::forWorkspace($this->workspace->id)
            ->orderByStatus()
            ->get();

        $this->assertEquals('Active plan', $plans[0]->title);
        $this->assertEquals('Draft plan', $plans[1]->title);
        $this->assertEquals('Archived plan', $plans[2]->title);
    }

    // =========================================================================
    // Tool Dependencies Tests
    // =========================================================================

    public function test_state_set_has_workspace_dependency(): void
    {
        $tool = new StateSet;
        $dependencies = $tool->dependencies();

        $this->assertNotEmpty($dependencies);
        $this->assertEquals('workspace_id', $dependencies[0]->key);
    }

    public function test_state_get_has_workspace_dependency(): void
    {
        $tool = new StateGet;
        $dependencies = $tool->dependencies();

        $this->assertNotEmpty($dependencies);
        $this->assertEquals('workspace_id', $dependencies[0]->key);
    }

    public function test_state_list_has_workspace_dependency(): void
    {
        $tool = new StateList;
        $dependencies = $tool->dependencies();

        $this->assertNotEmpty($dependencies);
        $this->assertEquals('workspace_id', $dependencies[0]->key);
    }

    public function test_plan_get_has_workspace_dependency(): void
    {
        $tool = new PlanGet;
        $dependencies = $tool->dependencies();

        $this->assertNotEmpty($dependencies);
        $this->assertEquals('workspace_id', $dependencies[0]->key);
    }

    public function test_plan_list_has_workspace_dependency(): void
    {
        $tool = new PlanList;
        $dependencies = $tool->dependencies();

        $this->assertNotEmpty($dependencies);
        $this->assertEquals('workspace_id', $dependencies[0]->key);
    }
}
