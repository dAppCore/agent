<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Sync;

use Core\Mod\Agentic\Actions\Sync\PushDispatchHistory;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\WorkspaceState;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class PushDispatchHistoryTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    private AgentPlan $plan;

    protected function setUp(): void
    {
        parent::setUp();

        Http::fake();

        $this->workspace = Workspace::factory()->create();
        $this->plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);
    }

    protected function tearDown(): void
    {
        Carbon::setTestNow();

        parent::tearDown();
    }

    public function test_PushDispatchHistory_handle_Good_updatesWorkspaceStateForAgentPlanId(): void
    {
        Carbon::setTestNow(Carbon::parse('2026-04-23T12:34:56+00:00'));

        $result = PushDispatchHistory::run($this->workspace->id, 'codex-agent', [[
            'repo' => 'dappco.re/go/agent',
            'workspace' => 'core-agent',
            'task' => 'Update workflow state after sync',
            'status' => 'completed',
            'agent_type' => 'codex',
            'agent_plan_id' => $this->plan->id,
            'findings' => [
                ['severity' => 'high'],
                ['severity' => 'medium'],
            ],
        ]]);

        $this->assertSame(['synced' => 1], $result);

        $states = WorkspaceState::forPlan($this->plan->id)->get()->keyBy('key');

        $this->assertCount(4, $states);
        $this->assertSyncState($states, 'sync.last_dispatch_at', '2026-04-23T12:34:56+00:00');
        $this->assertSyncState($states, 'sync.last_agent_type', 'codex');
        $this->assertSyncState($states, 'sync.last_findings_count', 2);
        $this->assertSyncState($states, 'sync.last_status', 'completed');
    }

    public function test_PushDispatchHistory_handle_Bad_resolvesPlanSlugForWorkspaceState(): void
    {
        Carbon::setTestNow(Carbon::parse('2026-04-23T13:45:00+00:00'));

        $result = PushDispatchHistory::run($this->workspace->id, 'claude-agent', [[
            'repo' => 'dappco.re/go/agent',
            'workspace' => 'core-agent',
            'task' => 'Resolve plan from slug',
            'status' => 'blocked',
            'agent_type' => 'claude',
            'plan_slug' => $this->plan->slug,
            'findings' => [
                ['severity' => 'low'],
            ],
        ]]);

        $this->assertSame(['synced' => 1], $result);

        $states = WorkspaceState::forPlan($this->plan)->get()->keyBy('key');

        $this->assertCount(4, $states);
        $this->assertSyncState($states, 'sync.last_dispatch_at', '2026-04-23T13:45:00+00:00');
        $this->assertSyncState($states, 'sync.last_agent_type', 'claude');
        $this->assertSyncState($states, 'sync.last_findings_count', 1);
        $this->assertSyncState($states, 'sync.last_status', 'blocked');
    }

    public function test_PushDispatchHistory_handle_Ugly_skipsWorkspaceStateWhenPlanIsMissing(): void
    {
        Carbon::setTestNow(Carbon::parse('2026-04-23T14:00:00+00:00'));

        $result = PushDispatchHistory::run($this->workspace->id, 'gemini-agent', [[
            'repo' => 'dappco.re/go/agent',
            'workspace' => 'core-agent',
            'task' => 'Dispatch without a matching plan',
            'status' => 'failed',
            'agent_type' => 'gemini',
            'plan_slug' => 'missing-plan',
            'findings' => [
                ['severity' => 'high'],
            ],
        ]]);

        $this->assertSame(['synced' => 1], $result);
        $this->assertSame(0, WorkspaceState::count());
    }

    /**
     * @param  Collection<string, WorkspaceState>  $states
     */
    private function assertSyncState(Collection $states, string $key, mixed $expectedValue): void
    {
        $state = $states->get($key);

        $this->assertInstanceOf(WorkspaceState::class, $state);
        $this->assertSame($expectedValue, $state->value);
        $this->assertSame(WorkspaceState::TYPE_JSON, $state->type);
        $this->assertSame('sync', $state->category);
        $this->assertNotNull($state->description);
    }
}
