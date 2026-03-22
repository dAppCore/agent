<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Actions\Issue\AddIssueComment;
use Core\Mod\Agentic\Actions\Issue\ArchiveIssue;
use Core\Mod\Agentic\Actions\Issue\CreateIssue;
use Core\Mod\Agentic\Actions\Issue\GetIssue;
use Core\Mod\Agentic\Actions\Issue\ListIssues;
use Core\Mod\Agentic\Actions\Issue\UpdateIssue;
use Core\Mod\Agentic\Models\Issue;
use Core\Mod\Agentic\Models\IssueComment;
use Core\Mod\Agentic\Models\Sprint;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class IssueTest extends TestCase
{
    use RefreshDatabase;

    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    // -- Model tests --

    public function test_issue_can_be_created(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'test-issue',
            'title' => 'Test Issue',
            'type' => Issue::TYPE_BUG,
            'status' => Issue::STATUS_OPEN,
            'priority' => Issue::PRIORITY_HIGH,
        ]);

        $this->assertDatabaseHas('issues', [
            'id' => $issue->id,
            'slug' => 'test-issue',
            'type' => 'bug',
            'priority' => 'high',
        ]);
    }

    public function test_issue_has_correct_default_status(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'defaults-test',
            'title' => 'Defaults',
        ]);

        $this->assertEquals(Issue::STATUS_OPEN, $issue->fresh()->status);
    }

    public function test_issue_can_be_closed(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'close-test',
            'title' => 'Close Me',
            'status' => Issue::STATUS_OPEN,
        ]);

        $issue->close();

        $fresh = $issue->fresh();
        $this->assertEquals(Issue::STATUS_CLOSED, $fresh->status);
        $this->assertNotNull($fresh->closed_at);
    }

    public function test_issue_can_be_reopened(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'reopen-test',
            'title' => 'Reopen Me',
            'status' => Issue::STATUS_CLOSED,
            'closed_at' => now(),
        ]);

        $issue->reopen();

        $fresh = $issue->fresh();
        $this->assertEquals(Issue::STATUS_OPEN, $fresh->status);
        $this->assertNull($fresh->closed_at);
    }

    public function test_issue_can_be_archived_with_reason(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'archive-test',
            'title' => 'Archive Me',
        ]);

        $issue->archive('Duplicate');

        $fresh = $issue->fresh();
        $this->assertEquals(Issue::STATUS_CLOSED, $fresh->status);
        $this->assertNotNull($fresh->archived_at);
        $this->assertEquals('Duplicate', $fresh->metadata['archive_reason']);
    }

    public function test_issue_generates_unique_slugs(): void
    {
        Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'my-issue',
            'title' => 'My Issue',
        ]);

        $slug = Issue::generateSlug('My Issue');

        $this->assertEquals('my-issue-1', $slug);
    }

    public function test_issue_belongs_to_sprint(): void
    {
        $sprint = Sprint::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'sprint-1',
            'title' => 'Sprint 1',
        ]);

        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'sprint_id' => $sprint->id,
            'slug' => 'sprint-issue',
            'title' => 'Sprint Issue',
        ]);

        $this->assertEquals($sprint->id, $issue->sprint->id);
    }

    public function test_issue_has_comments(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'commented',
            'title' => 'Has Comments',
        ]);

        IssueComment::create([
            'issue_id' => $issue->id,
            'author' => 'claude',
            'body' => 'Investigating.',
        ]);

        $this->assertCount(1, $issue->fresh()->comments);
    }

    public function test_issue_label_management(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'label-test',
            'title' => 'Labels',
            'labels' => [],
        ]);

        $issue->addLabel('agentic');
        $this->assertContains('agentic', $issue->fresh()->labels);

        $issue->addLabel('agentic'); // duplicate — should not add
        $this->assertCount(1, $issue->fresh()->labels);

        $issue->removeLabel('agentic');
        $this->assertEmpty($issue->fresh()->labels);
    }

    public function test_issue_scopes(): void
    {
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'open-1', 'title' => 'A', 'status' => Issue::STATUS_OPEN]);
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'ip-1', 'title' => 'B', 'status' => Issue::STATUS_IN_PROGRESS]);
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'closed-1', 'title' => 'C', 'status' => Issue::STATUS_CLOSED]);

        $this->assertCount(1, Issue::open()->get());
        $this->assertCount(1, Issue::inProgress()->get());
        $this->assertCount(1, Issue::closed()->get());
        $this->assertCount(2, Issue::notClosed()->get());
    }

    public function test_issue_to_mcp_context(): void
    {
        $issue = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'mcp-test',
            'title' => 'MCP Context',
            'type' => Issue::TYPE_FEATURE,
        ]);

        $context = $issue->toMcpContext();

        $this->assertIsArray($context);
        $this->assertEquals('mcp-test', $context['slug']);
        $this->assertEquals('feature', $context['type']);
        $this->assertArrayHasKey('status', $context);
        $this->assertArrayHasKey('priority', $context);
        $this->assertArrayHasKey('labels', $context);
    }

    // -- Action tests --

    public function test_create_issue_action(): void
    {
        $issue = CreateIssue::run([
            'title' => 'New Bug',
            'type' => 'bug',
            'priority' => 'high',
            'labels' => ['agentic'],
        ], $this->workspace->id);

        $this->assertInstanceOf(Issue::class, $issue);
        $this->assertEquals('New Bug', $issue->title);
        $this->assertEquals('bug', $issue->type);
        $this->assertEquals('high', $issue->priority);
        $this->assertEquals(Issue::STATUS_OPEN, $issue->status);
    }

    public function test_create_issue_action_validates_title(): void
    {
        $this->expectException(\InvalidArgumentException::class);

        CreateIssue::run(['title' => ''], $this->workspace->id);
    }

    public function test_create_issue_action_validates_type(): void
    {
        $this->expectException(\InvalidArgumentException::class);

        CreateIssue::run([
            'title' => 'Bad Type',
            'type' => 'invalid',
        ], $this->workspace->id);
    }

    public function test_get_issue_action(): void
    {
        $created = Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'get-test',
            'title' => 'Get Me',
        ]);

        $found = GetIssue::run('get-test', $this->workspace->id);

        $this->assertEquals($created->id, $found->id);
    }

    public function test_get_issue_action_throws_for_missing(): void
    {
        $this->expectException(\InvalidArgumentException::class);

        GetIssue::run('nonexistent', $this->workspace->id);
    }

    public function test_list_issues_action(): void
    {
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'list-1', 'title' => 'A', 'status' => Issue::STATUS_OPEN]);
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'list-2', 'title' => 'B', 'status' => Issue::STATUS_CLOSED]);

        $all = ListIssues::run($this->workspace->id, includeClosed: true);
        $this->assertCount(2, $all);

        $open = ListIssues::run($this->workspace->id);
        $this->assertCount(1, $open);
    }

    public function test_list_issues_filters_by_type(): void
    {
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'bug-1', 'title' => 'Bug', 'type' => Issue::TYPE_BUG]);
        Issue::create(['workspace_id' => $this->workspace->id, 'slug' => 'feat-1', 'title' => 'Feature', 'type' => Issue::TYPE_FEATURE]);

        $bugs = ListIssues::run($this->workspace->id, type: 'bug');
        $this->assertCount(1, $bugs);
        $this->assertEquals('bug', $bugs->first()->type);
    }

    public function test_update_issue_action(): void
    {
        Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'update-test',
            'title' => 'Update Me',
            'status' => Issue::STATUS_OPEN,
        ]);

        $updated = UpdateIssue::run('update-test', [
            'status' => Issue::STATUS_IN_PROGRESS,
            'priority' => Issue::PRIORITY_URGENT,
        ], $this->workspace->id);

        $this->assertEquals(Issue::STATUS_IN_PROGRESS, $updated->status);
        $this->assertEquals(Issue::PRIORITY_URGENT, $updated->priority);
    }

    public function test_update_issue_sets_closed_at_when_closing(): void
    {
        Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'close-via-update',
            'title' => 'Close Me',
            'status' => Issue::STATUS_OPEN,
        ]);

        $updated = UpdateIssue::run('close-via-update', [
            'status' => Issue::STATUS_CLOSED,
        ], $this->workspace->id);

        $this->assertNotNull($updated->closed_at);
    }

    public function test_archive_issue_action(): void
    {
        Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'archive-action',
            'title' => 'Archive Me',
        ]);

        $archived = ArchiveIssue::run('archive-action', $this->workspace->id, 'Not needed');

        $this->assertNotNull($archived->archived_at);
        $this->assertEquals(Issue::STATUS_CLOSED, $archived->status);
    }

    public function test_add_issue_comment_action(): void
    {
        Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'comment-action',
            'title' => 'Comment On Me',
        ]);

        $comment = AddIssueComment::run(
            'comment-action',
            $this->workspace->id,
            'gemini',
            'Found the root cause.',
        );

        $this->assertInstanceOf(IssueComment::class, $comment);
        $this->assertEquals('gemini', $comment->author);
        $this->assertEquals('Found the root cause.', $comment->body);
    }

    public function test_add_comment_validates_empty_body(): void
    {
        Issue::create([
            'workspace_id' => $this->workspace->id,
            'slug' => 'empty-comment',
            'title' => 'Empty Comment',
        ]);

        $this->expectException(\InvalidArgumentException::class);

        AddIssueComment::run('empty-comment', $this->workspace->id, 'claude', '');
    }
}
