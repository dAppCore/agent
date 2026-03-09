<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AgentSessionTest extends TestCase
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
        $session = AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $this->assertDatabaseHas('agent_sessions', [
            'id' => $session->id,
            'workspace_id' => $this->workspace->id,
        ]);
    }

    public function test_it_can_be_started_statically(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session = AgentSession::start($plan, AgentSession::AGENT_OPUS);

        $this->assertDatabaseHas('agent_sessions', ['id' => $session->id]);
        $this->assertEquals($plan->id, $session->agent_plan_id);
        $this->assertEquals($this->workspace->id, $session->workspace_id);
        $this->assertEquals(AgentSession::AGENT_OPUS, $session->agent_type);
        $this->assertEquals(AgentSession::STATUS_ACTIVE, $session->status);
        $this->assertStringStartsWith('sess_', $session->session_id);
    }

    public function test_status_helper_methods(): void
    {
        $active = AgentSession::factory()->active()->make();
        $paused = AgentSession::factory()->paused()->make();
        $completed = AgentSession::factory()->completed()->make();
        $failed = AgentSession::factory()->failed()->make();

        $this->assertTrue($active->isActive());
        $this->assertTrue($paused->isPaused());
        $this->assertTrue($completed->isEnded());
        $this->assertTrue($failed->isEnded());

        $this->assertFalse($active->isEnded());
        $this->assertFalse($paused->isEnded());
    }

    public function test_it_can_be_paused(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->pause();

        $this->assertTrue($session->fresh()->isPaused());
    }

    public function test_it_can_be_resumed(): void
    {
        $session = AgentSession::factory()->paused()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->resume();

        $fresh = $session->fresh();
        $this->assertTrue($fresh->isActive());
        $this->assertNotNull($fresh->last_active_at);
    }

    public function test_it_can_be_completed_with_summary(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->complete('All tasks finished successfully');

        $fresh = $session->fresh();
        $this->assertEquals(AgentSession::STATUS_COMPLETED, $fresh->status);
        $this->assertEquals('All tasks finished successfully', $fresh->final_summary);
        $this->assertNotNull($fresh->ended_at);
    }

    public function test_it_can_fail_with_reason(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->fail('Error occurred');

        $fresh = $session->fresh();
        $this->assertEquals(AgentSession::STATUS_FAILED, $fresh->status);
        $this->assertEquals('Error occurred', $fresh->final_summary);
        $this->assertNotNull($fresh->ended_at);
    }

    public function test_it_logs_actions(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'work_log' => [],
        ]);

        $session->logAction('Created file', ['path' => '/test.php']);

        $log = $session->fresh()->work_log;
        $this->assertCount(1, $log);
        $this->assertEquals('Created file', $log[0]['action']);
        $this->assertEquals(['path' => '/test.php'], $log[0]['details']);
        $this->assertNotNull($log[0]['timestamp']);
    }

    public function test_it_adds_typed_work_log_entries(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'work_log' => [],
        ]);

        $session->addWorkLogEntry('Task completed', 'success', ['task' => 'build']);

        $log = $session->fresh()->work_log;
        $this->assertEquals('Task completed', $log[0]['message']);
        $this->assertEquals('success', $log[0]['type']);
        $this->assertEquals(['task' => 'build'], $log[0]['data']);
    }

    public function test_it_gets_recent_actions(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'work_log' => [],
        ]);

        for ($i = 1; $i <= 15; $i++) {
            $session->logAction("Action {$i}");
        }

        $recent = $session->fresh()->getRecentActions(5);

        $this->assertCount(5, $recent);
        $this->assertEquals('Action 15', $recent[0]['action']);
        $this->assertEquals('Action 11', $recent[4]['action']);
    }

    public function test_it_adds_artifacts(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'artifacts' => [],
        ]);

        $session->addArtifact('/app/Test.php', 'created', ['lines' => 50]);

        $artifacts = $session->fresh()->artifacts;
        $this->assertCount(1, $artifacts);
        $this->assertEquals('/app/Test.php', $artifacts[0]['path']);
        $this->assertEquals('created', $artifacts[0]['action']);
        $this->assertEquals(['lines' => 50], $artifacts[0]['metadata']);
    }

    public function test_it_filters_artifacts_by_action(): void
    {
        $session = AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
            'artifacts' => [
                ['path' => '/file1.php', 'action' => 'created'],
                ['path' => '/file2.php', 'action' => 'modified'],
                ['path' => '/file3.php', 'action' => 'created'],
            ],
        ]);

        $created = $session->getArtifactsByAction('created');

        $this->assertCount(2, $created);
    }

    public function test_it_updates_context_summary(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->updateContextSummary(['current_task' => 'testing', 'progress' => 50]);

        $this->assertEquals(
            ['current_task' => 'testing', 'progress' => 50],
            $session->fresh()->context_summary
        );
    }

    public function test_it_adds_to_context(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'context_summary' => ['existing' => 'value'],
        ]);

        $session->addToContext('new_key', 'new_value');

        $context = $session->fresh()->context_summary;
        $this->assertEquals('value', $context['existing']);
        $this->assertEquals('new_value', $context['new_key']);
    }

    public function test_it_prepares_handoff(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->prepareHandoff(
            'Completed phase 1',
            ['Continue with phase 2'],
            ['Needs API key'],
            ['important' => 'data']
        );

        $fresh = $session->fresh();
        $this->assertTrue($fresh->isPaused());
        $this->assertEquals('Completed phase 1', $fresh->handoff_notes['summary']);
        $this->assertEquals(['Continue with phase 2'], $fresh->handoff_notes['next_steps']);
        $this->assertEquals(['Needs API key'], $fresh->handoff_notes['blockers']);
        $this->assertEquals(['important' => 'data'], $fresh->handoff_notes['context_for_next']);
    }

    public function test_it_gets_handoff_context(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Test Plan',
        ]);

        $session = AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
            'agent_plan_id' => $plan->id,
            'context_summary' => ['test' => 'data'],
        ]);

        $context = $session->getHandoffContext();

        $this->assertArrayHasKey('session_id', $context);
        $this->assertArrayHasKey('agent_type', $context);
        $this->assertArrayHasKey('context_summary', $context);
        $this->assertArrayHasKey('plan', $context);
        $this->assertEquals('Test Plan', $context['plan']['title']);
    }

    public function test_it_calculates_duration(): void
    {
        $session = AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
            'started_at' => now()->subMinutes(90),
            'ended_at' => now(),
        ]);

        $this->assertEquals(90, $session->getDuration());
        $this->assertEquals('1h 30m', $session->getDurationFormatted());
    }

    public function test_duration_for_short_sessions(): void
    {
        $session = AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
            'started_at' => now()->subMinutes(30),
            'ended_at' => now(),
        ]);

        $this->assertEquals('30m', $session->getDurationFormatted());
    }

    public function test_duration_uses_now_when_not_ended(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'started_at' => now()->subMinutes(10),
            'ended_at' => null,
        ]);

        $this->assertEquals(10, $session->getDuration());
    }

    public function test_active_scope(): void
    {
        AgentSession::factory()->active()->create(['workspace_id' => $this->workspace->id]);
        AgentSession::factory()->paused()->create(['workspace_id' => $this->workspace->id]);
        AgentSession::factory()->completed()->create(['workspace_id' => $this->workspace->id]);

        $active = AgentSession::active()->get();

        $this->assertCount(1, $active);
    }

    public function test_for_plan_scope(): void
    {
        $plan = AgentPlan::factory()->create(['workspace_id' => $this->workspace->id]);
        AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
            'agent_plan_id' => $plan->id,
        ]);
        AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
            'agent_plan_id' => null,
        ]);

        $sessions = AgentSession::forPlan($plan)->get();

        $this->assertCount(1, $sessions);
    }

    public function test_agent_type_factory_states(): void
    {
        $opus = AgentSession::factory()->opus()->make();
        $sonnet = AgentSession::factory()->sonnet()->make();
        $haiku = AgentSession::factory()->haiku()->make();

        $this->assertEquals(AgentSession::AGENT_OPUS, $opus->agent_type);
        $this->assertEquals(AgentSession::AGENT_SONNET, $sonnet->agent_type);
        $this->assertEquals(AgentSession::AGENT_HAIKU, $haiku->agent_type);
    }

    public function test_for_plan_factory_state(): void
    {
        $plan = AgentPlan::factory()->create(['workspace_id' => $this->workspace->id]);

        $session = AgentSession::factory()->forPlan($plan)->create();

        $this->assertEquals($plan->id, $session->agent_plan_id);
        $this->assertEquals($plan->workspace_id, $session->workspace_id);
    }

    public function test_to_mcp_context_returns_array(): void
    {
        $session = AgentSession::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $context = $session->toMcpContext();

        $this->assertIsArray($context);
        $this->assertArrayHasKey('session_id', $context);
        $this->assertArrayHasKey('agent_type', $context);
        $this->assertArrayHasKey('status', $context);
        $this->assertArrayHasKey('duration', $context);
        $this->assertArrayHasKey('action_count', $context);
        $this->assertArrayHasKey('artifact_count', $context);
    }

    public function test_touch_activity_updates_timestamp(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
            'last_active_at' => now()->subHour(),
        ]);

        $oldTime = $session->last_active_at;
        $session->touchActivity();

        $this->assertGreaterThan($oldTime, $session->fresh()->last_active_at);
    }

    public function test_end_with_status(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->end(AgentSession::STATUS_COMPLETED, 'Done');

        $fresh = $session->fresh();
        $this->assertEquals(AgentSession::STATUS_COMPLETED, $fresh->status);
        $this->assertEquals('Done', $fresh->final_summary);
        $this->assertNotNull($fresh->ended_at);
    }

    public function test_end_defaults_to_completed_for_invalid_status(): void
    {
        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $session->end('invalid_status');

        $this->assertEquals(AgentSession::STATUS_COMPLETED, $session->fresh()->status);
    }
}
