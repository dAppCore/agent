<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class PlanRetentionTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    public function test_cleanup_permanently_deletes_archived_plans_past_retention(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(91),
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 90])
            ->assertSuccessful();

        $this->assertNull(AgentPlan::withTrashed()->find($plan->id));
    }

    public function test_cleanup_keeps_recently_archived_plans(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(10),
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 90])
            ->assertSuccessful();

        $this->assertNotNull(AgentPlan::find($plan->id));
    }

    public function test_cleanup_does_not_affect_non_archived_plans(): void
    {
        $active = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ACTIVE,
        ]);

        $draft = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_DRAFT,
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 1])
            ->assertSuccessful();

        $this->assertNotNull(AgentPlan::find($active->id));
        $this->assertNotNull(AgentPlan::find($draft->id));
    }

    public function test_cleanup_skips_archived_plans_without_archived_at(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => null,
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 1])
            ->assertSuccessful();

        $this->assertNotNull(AgentPlan::find($plan->id));
    }

    public function test_dry_run_does_not_delete_plans(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(100),
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 90, '--dry-run' => true])
            ->assertSuccessful();

        $this->assertNotNull(AgentPlan::find($plan->id));
    }

    public function test_dry_run_reports_count(): void
    {
        AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(100),
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 90, '--dry-run' => true])
            ->expectsOutputToContain('DRY RUN')
            ->assertSuccessful();
    }

    public function test_cleanup_disabled_when_days_is_zero(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(1000),
        ]);

        $this->artisan('agentic:plan-cleanup', ['--days' => 0])
            ->assertSuccessful();

        $this->assertNotNull(AgentPlan::find($plan->id));
    }

    public function test_uses_config_retention_days_by_default(): void
    {
        config(['agentic.plan_retention_days' => 30]);

        $old = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(31),
        ]);

        $recent = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'status' => AgentPlan::STATUS_ARCHIVED,
            'archived_at' => now()->subDays(10),
        ]);

        $this->artisan('agentic:plan-cleanup')
            ->assertSuccessful();

        $this->assertNull(AgentPlan::withTrashed()->find($old->id));
        $this->assertNotNull(AgentPlan::find($recent->id));
    }

    public function test_archive_sets_archived_at(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $this->assertNull($plan->archived_at);

        $plan->archive('test reason');

        $fresh = $plan->fresh();
        $this->assertEquals(AgentPlan::STATUS_ARCHIVED, $fresh->status);
        $this->assertNotNull($fresh->archived_at);
        $this->assertEquals('test reason', $fresh->metadata['archive_reason']);
    }

    public function test_archive_without_reason_still_sets_archived_at(): void
    {
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $plan->archive();

        $fresh = $plan->fresh();
        $this->assertEquals(AgentPlan::STATUS_ARCHIVED, $fresh->status);
        $this->assertNotNull($fresh->archived_at);
    }
}
