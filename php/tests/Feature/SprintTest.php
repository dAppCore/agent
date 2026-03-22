<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Actions\Sprint\ArchiveSprint;
use Core\Mod\Agentic\Actions\Sprint\CreateSprint;
use Core\Mod\Agentic\Actions\Sprint\GetSprint;
use Core\Mod\Agentic\Actions\Sprint\ListSprints;
use Core\Mod\Agentic\Actions\Sprint\UpdateSprint;
use Core\Mod\Agentic\Models\Issue;
use Core\Mod\Agentic\Models\Sprint;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class SprintTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    // -- Model tests --

    public function test_sprint_can_be_created(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'sprint-1',
            'title' => 'Sprint 1',
            'goal' => 'Ship MVP',
        ]);

        $this->assertDatabaseHas('sprints', [
            'id' => $sprint->id,
            'slug' => 'sprint-1',
            'goal' => 'Ship MVP',
        ]);
    }

    public function test_sprint_has_default_planning_status(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'defaults',
            'title' => 'Defaults',
        ]);

        $this->assertEquals(Sprint::STATUS_PLANNING, $sprint->fresh()->status);
    }

    public function test_sprint_can_be_activated(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'activate-test',
            'title' => 'Activate',
            'status' => Sprint::STATUS_PLANNING,
        ]);

        $sprint->activate();

        $fresh = $sprint->fresh();
        $this->assertEquals(Sprint::STATUS_ACTIVE, $fresh->status);
        $this->assertNotNull($fresh->started_at);
    }

    public function test_sprint_can_be_completed(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'complete-test',
            'title' => 'Complete',
            'status' => Sprint::STATUS_ACTIVE,
        ]);

        $sprint->complete();

        $fresh = $sprint->fresh();
        $this->assertEquals(Sprint::STATUS_COMPLETED, $fresh->status);
        $this->assertNotNull($fresh->ended_at);
    }

    public function test_sprint_can_be_cancelled_with_reason(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'cancel-test',
            'title' => 'Cancel',
        ]);

        $sprint->cancel('Scope changed');

        $fresh = $sprint->fresh();
        $this->assertEquals(Sprint::STATUS_CANCELLED, $fresh->status);
        $this->assertNotNull($fresh->ended_at);
        $this->assertNotNull($fresh->archived_at);
        $this->assertEquals('Scope changed', $fresh->metadata['cancel_reason']);
    }

    public function test_sprint_generates_unique_slugs(): void
    {
        Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'sprint-1',
            'title' => 'Sprint 1',
        ]);

        $slug = Sprint::generateSlug('Sprint 1');

        $this->assertEquals('sprint-1-1', $slug);
    }

    public function test_sprint_has_issues(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'with-issues',
            'title' => 'Has Issues',
        ]);

        Issue::create([
            'workspace_id' => $this->workspace->id,
            'sprint_id' => $sprint->id,
            'slug' => 'issue-1',
            'title' => 'First',
        ]);

        $this->assertCount(1, $sprint->fresh()->issues);
    }

    public function test_sprint_calculates_progress(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'progress-test',
            'title' => 'Progress',
        ]);

        Issue::create(['workspace_id' => $this->workspace->id, 'sprint_id' => $sprint->id, 'slug' => 'p-1', 'title' => 'A', 'status' => Issue::STATUS_OPEN]);
        Issue::create(['workspace_id' => $this->workspace->id, 'sprint_id' => $sprint->id, 'slug' => 'p-2', 'title' => 'B', 'status' => Issue::STATUS_IN_PROGRESS]);
        Issue::create(['workspace_id' => $this->workspace->id, 'sprint_id' => $sprint->id, 'slug' => 'p-3', 'title' => 'C', 'status' => Issue::STATUS_CLOSED]);
        Issue::create(['workspace_id' => $this->workspace->id, 'sprint_id' => $sprint->id, 'slug' => 'p-4', 'title' => 'D', 'status' => Issue::STATUS_CLOSED]);

        $progress = $sprint->getProgress();

        $this->assertEquals(4, $progress['total']);
        $this->assertEquals(2, $progress['closed']);
        $this->assertEquals(1, $progress['in_progress']);
        $this->assertEquals(1, $progress['open']);
        $this->assertEquals(50, $progress['percentage']);
    }

    public function test_sprint_scopes(): void
    {
        Sprint::create(['workspace_id' => $this->workspace->id, 'slug' => 's-1', 'title' => 'A', 'status' => Sprint::STATUS_PLANNING]);
        Sprint::create(['workspace_id' => $this->workspace->id, 'slug' => 's-2', 'title' => 'B', 'status' => Sprint::STATUS_ACTIVE]);
        Sprint::create(['workspace_id' => $this->workspace->id, 'slug' => 's-3', 'title' => 'C', 'status' => Sprint::STATUS_CANCELLED]);

        $this->assertCount(1, Sprint::active()->get());
        $this->assertCount(1, Sprint::planning()->get());
        $this->assertCount(2, Sprint::notCancelled()->get());
    }

    public function test_sprint_to_mcp_context(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'mcp-test',
            'title' => 'MCP Context',
            'goal' => 'Test goal',
        ]);

        $context = $sprint->toMcpContext();

        $this->assertIsArray($context);
        $this->assertEquals('mcp-test', $context['slug']);
        $this->assertEquals('Test goal', $context['goal']);
        $this->assertArrayHasKey('status', $context);
        $this->assertArrayHasKey('progress', $context);
    }

    // -- Action tests --

    public function test_create_sprint_action(): void
    {
        $sprint = CreateSprint::run([
            'title' => 'New Sprint',
            'goal' => 'Deliver features',
        ], $this->workspace->id);

        $this->assertInstanceOf(Sprint::class, $sprint);
        $this->assertEquals('New Sprint', $sprint->title);
        $this->assertEquals(Sprint::STATUS_PLANNING, $sprint->status);
    }

    public function test_create_sprint_validates_title(): void
    {
        $this->expectException(\InvalidArgumentException::class);

        CreateSprint::run(['title' => ''], $this->workspace->id);
    }

    public function test_get_sprint_action(): void
    {
        $created = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'get-test',
            'title' => 'Get Me',
        ]);

        $found = GetSprint::run('get-test', $this->workspace->id);

        $this->assertEquals($created->id, $found->id);
    }

    public function test_get_sprint_throws_for_missing(): void
    {
        $this->expectException(\InvalidArgumentException::class);

        GetSprint::run('nonexistent', $this->workspace->id);
    }

    public function test_list_sprints_action(): void
    {
        Sprint::create(['workspace_id' => $this->workspace->id, 'slug' => 'ls-1', 'title' => 'A', 'status' => Sprint::STATUS_ACTIVE]);
        Sprint::create(['workspace_id' => $this->workspace->id, 'slug' => 'ls-2', 'title' => 'B', 'status' => Sprint::STATUS_CANCELLED]);

        $all = ListSprints::run($this->workspace->id, includeCancelled: true);
        $this->assertCount(2, $all);

        $notCancelled = ListSprints::run($this->workspace->id);
        $this->assertCount(1, $notCancelled);
    }

    public function test_update_sprint_action(): void
    {
        Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'update-test',
            'title' => 'Update Me',
            'status' => Sprint::STATUS_PLANNING,
        ]);

        $updated = UpdateSprint::run('update-test', [
            'status' => Sprint::STATUS_ACTIVE,
        ], $this->workspace->id);

        $this->assertEquals(Sprint::STATUS_ACTIVE, $updated->status);
        $this->assertNotNull($updated->started_at);
    }

    public function test_update_sprint_validates_status(): void
    {
        Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'bad-status',
            'title' => 'Bad',
        ]);

        $this->expectException(\InvalidArgumentException::class);

        UpdateSprint::run('bad-status', ['status' => 'invalid'], $this->workspace->id);
    }

    public function test_archive_sprint_action(): void
    {
        Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'archive-test',
            'title' => 'Archive Me',
        ]);

        $archived = ArchiveSprint::run('archive-test', $this->workspace->id, 'Done');

        $this->assertEquals(Sprint::STATUS_CANCELLED, $archived->status);
        $this->assertNotNull($archived->archived_at);
    }
}
