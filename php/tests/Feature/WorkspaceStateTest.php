<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\WorkspaceState;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Tests for WorkspaceState model (consolidated from WorkspaceState + AgentWorkspaceState).
 */
class WorkspaceStateTest extends TestCase
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

    // =========================================================================
    // Table and fillable
    // =========================================================================

    public function test_it_uses_agent_workspace_states_table(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'test_key',
            'value' => ['data' => 'value'],
            'type' => WorkspaceState::TYPE_JSON,
        ]);

        $this->assertDatabaseHas('agent_workspace_states', [
            'id' => $state->id,
            'key' => 'test_key',
        ]);
    }

    public function test_it_casts_value_as_array(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'array_key',
            'value' => ['foo' => 'bar', 'count' => 42],
        ]);

        $fresh = $state->fresh();
        $this->assertIsArray($fresh->value);
        $this->assertEquals('bar', $fresh->value['foo']);
        $this->assertEquals(42, $fresh->value['count']);
    }

    // =========================================================================
    // Type constants and helpers
    // =========================================================================

    public function test_type_constants_are_defined(): void
    {
        $this->assertEquals('json', WorkspaceState::TYPE_JSON);
        $this->assertEquals('markdown', WorkspaceState::TYPE_MARKDOWN);
        $this->assertEquals('code', WorkspaceState::TYPE_CODE);
        $this->assertEquals('reference', WorkspaceState::TYPE_REFERENCE);
    }

    public function test_isJson_returns_true_for_json_type(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'json_key',
            'value' => ['x' => 1],
            'type' => WorkspaceState::TYPE_JSON,
        ]);

        $this->assertTrue($state->isJson());
        $this->assertFalse($state->isMarkdown());
        $this->assertFalse($state->isCode());
        $this->assertFalse($state->isReference());
    }

    public function test_isMarkdown_returns_true_for_markdown_type(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'md_key',
            'value' => null,
            'type' => WorkspaceState::TYPE_MARKDOWN,
        ]);

        $this->assertTrue($state->isMarkdown());
        $this->assertFalse($state->isJson());
    }

    public function test_getFormattedValue_returns_json_string(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'fmt_key',
            'value' => ['a' => 1],
            'type' => WorkspaceState::TYPE_JSON,
        ]);

        $formatted = $state->getFormattedValue();
        $this->assertIsString($formatted);
        $this->assertStringContainsString('"a"', $formatted);
    }

    // =========================================================================
    // Relationship
    // =========================================================================

    public function test_it_belongs_to_plan(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'rel_key',
            'value' => [],
        ]);

        $this->assertEquals($this->plan->id, $state->plan->id);
    }

    public function test_plan_has_many_states(): void
    {
        WorkspaceState::create(['agent_plan_id' => $this->plan->id, 'key' => 'k1', 'value' => []]);
        WorkspaceState::create(['agent_plan_id' => $this->plan->id, 'key' => 'k2', 'value' => []]);

        $this->assertCount(2, $this->plan->states);
    }

    // =========================================================================
    // Scopes
    // =========================================================================

    public function test_scopeForPlan_filters_by_plan_id(): void
    {
        $otherPlan = AgentPlan::factory()->create(['workspace_id' => $this->workspace->id]);

        WorkspaceState::create(['agent_plan_id' => $this->plan->id, 'key' => 'mine', 'value' => []]);
        WorkspaceState::create(['agent_plan_id' => $otherPlan->id, 'key' => 'theirs', 'value' => []]);

        $results = WorkspaceState::forPlan($this->plan)->get();

        $this->assertCount(1, $results);
        $this->assertEquals('mine', $results->first()->key);
    }

    public function test_scopeForPlan_accepts_int(): void
    {
        WorkspaceState::create(['agent_plan_id' => $this->plan->id, 'key' => 'int_scope', 'value' => []]);

        $results = WorkspaceState::forPlan($this->plan->id)->get();

        $this->assertCount(1, $results);
    }

    public function test_scopeOfType_filters_by_type(): void
    {
        WorkspaceState::create(['agent_plan_id' => $this->plan->id, 'key' => 'j', 'value' => [], 'type' => WorkspaceState::TYPE_JSON]);
        WorkspaceState::create(['agent_plan_id' => $this->plan->id, 'key' => 'm', 'value' => null, 'type' => WorkspaceState::TYPE_MARKDOWN]);

        $jsonStates = WorkspaceState::ofType(WorkspaceState::TYPE_JSON)->get();

        $this->assertCount(1, $jsonStates);
        $this->assertEquals('j', $jsonStates->first()->key);
    }

    // =========================================================================
    // Static helpers
    // =========================================================================

    public function test_getValue_returns_stored_value(): void
    {
        WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'endpoints',
            'value' => ['count' => 12],
        ]);

        $value = WorkspaceState::getValue($this->plan, 'endpoints');

        $this->assertEquals(['count' => 12], $value);
    }

    public function test_getValue_returns_default_when_key_missing(): void
    {
        $value = WorkspaceState::getValue($this->plan, 'nonexistent', 'default_val');

        $this->assertEquals('default_val', $value);
    }

    public function test_setValue_creates_new_state(): void
    {
        $state = WorkspaceState::setValue($this->plan, 'api_findings', ['endpoints' => 5]);

        $this->assertDatabaseHas('agent_workspace_states', [
            'agent_plan_id' => $this->plan->id,
            'key' => 'api_findings',
        ]);
        $this->assertEquals(['endpoints' => 5], $state->value);
    }

    public function test_setValue_updates_existing_state(): void
    {
        WorkspaceState::setValue($this->plan, 'counter', ['n' => 1]);
        WorkspaceState::setValue($this->plan, 'counter', ['n' => 2]);

        $this->assertDatabaseCount('agent_workspace_states', 1);
        $this->assertEquals(['n' => 2], WorkspaceState::getValue($this->plan, 'counter'));
    }

    // =========================================================================
    // MCP output
    // =========================================================================

    public function test_toMcpContext_returns_expected_keys(): void
    {
        $state = WorkspaceState::create([
            'agent_plan_id' => $this->plan->id,
            'key' => 'mcp_key',
            'value' => ['x' => 99],
            'type' => WorkspaceState::TYPE_JSON,
            'description' => 'Test state entry',
        ]);

        $context = $state->toMcpContext();

        $this->assertArrayHasKey('key', $context);
        $this->assertArrayHasKey('type', $context);
        $this->assertArrayHasKey('description', $context);
        $this->assertArrayHasKey('value', $context);
        $this->assertArrayHasKey('updated_at', $context);
        $this->assertEquals('mcp_key', $context['key']);
        $this->assertEquals('Test state entry', $context['description']);
    }

    // =========================================================================
    // Plan setState() integration
    // =========================================================================

    public function test_plan_setState_creates_workspace_state(): void
    {
        $state = $this->plan->setState('progress', ['done' => 3, 'total' => 10]);

        $this->assertInstanceOf(WorkspaceState::class, $state);
        $this->assertEquals('progress', $state->key);
        $this->assertEquals(['done' => 3, 'total' => 10], $state->value);
    }

    public function test_plan_getState_retrieves_value(): void
    {
        $this->plan->setState('status_data', ['phase' => 'analysis']);

        $value = $this->plan->getState('status_data');

        $this->assertEquals(['phase' => 'analysis'], $value);
    }
}
