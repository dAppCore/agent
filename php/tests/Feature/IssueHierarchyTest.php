<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Issue\CreateIssue;
use Core\Mod\Agentic\Actions\Issue\SplitIssue;
use Core\Mod\Agentic\Actions\Issue\UpdateIssue;
use Core\Mod\Agentic\Models\Issue;
use Core\Tenant\Models\Workspace;

beforeEach(function () {
    $this->workspace = Workspace::factory()->create();
});

// ─── Epics ───────────────────────────────────────────────────────────────

it('creates an epic through the action', function () {
    $epic = CreateIssue::run([
        'title' => 'Retire go.work',
        'type' => Issue::TYPE_EPIC,
    ], $this->workspace->id);

    expect($epic->type)->toBe(Issue::TYPE_EPIC)
        ->and($epic->isEpic())->toBeTrue()
        ->and($epic->isSubtask())->toBeFalse();
});

it('rejects an unknown type', function () {
    CreateIssue::run(['title' => 'Nope', 'type' => 'saga'], $this->workspace->id);
})->throws(InvalidArgumentException::class, 'type must be one of');

// ─── Splitting ───────────────────────────────────────────────────────────

it('splits an epic into children that inherit its context', function () {
    $epic = CreateIssue::run([
        'title' => 'Retire go.work',
        'type' => Issue::TYPE_EPIC,
        'repository' => 'dappcore/go-inference',
    ], $this->workspace->id);

    $children = SplitIssue::run($epic, [
        ['title' => 'Map the dependency layers', 'discipline' => Issue::DISCIPLINE_PLANNING],
        ['title' => 'Tag the L0 modules', 'discipline' => Issue::DISCIPLINE_CODE, 'priority' => Issue::PRIORITY_HIGH],
        ['title' => 'Rewrite the release guide', 'discipline' => Issue::DISCIPLINE_CONTENT],
    ]);

    expect($children)->toHaveCount(3)
        ->and($epic->children()->count())->toBe(3);

    // A chunk supplied only a title and a discipline; workspace and repository
    // came down from the parent.
    $planning = $children->firstWhere('discipline', Issue::DISCIPLINE_PLANNING);
    expect($planning->repository)->toBe('dappcore/go-inference')
        ->and($planning->workspace_id)->toBe($this->workspace->id)
        ->and($planning->parent_id)->toBe($epic->id)
        ->and($planning->isSubtask())->toBeTrue()
        ->and($planning->type)->toBe(Issue::TYPE_TASK);
});

it('nests subtasks under a subtask', function () {
    $epic = CreateIssue::run(['title' => 'Programme', 'type' => Issue::TYPE_EPIC], $this->workspace->id);
    $task = SplitIssue::run($epic, [['title' => 'Big chunk']])->first();
    $subs = SplitIssue::run($task, [['title' => 'Smaller bit'], ['title' => 'Other bit']]);

    expect($subs)->toHaveCount(2)
        ->and($subs->first()->parent_id)->toBe($task->id)
        ->and($task->parent_id)->toBe($epic->id)
        // The epic counts only its direct children, so a grandchild does not
        // inflate its progress.
        ->and($epic->getProgress()['total'])->toBe(1);
});

it('refuses to split with no chunks', function () {
    $epic = CreateIssue::run(['title' => 'Empty', 'type' => Issue::TYPE_EPIC], $this->workspace->id);
    SplitIssue::run($epic, []);
})->throws(InvalidArgumentException::class, 'At least one chunk is required');

it('refuses a chunk with no title, creating none of them', function () {
    $epic = CreateIssue::run(['title' => 'Partial', 'type' => Issue::TYPE_EPIC], $this->workspace->id);

    expect(fn () => SplitIssue::run($epic, [
        ['title' => 'Fine'],
        ['discipline' => Issue::DISCIPLINE_CODE],
    ]))->toThrow(InvalidArgumentException::class, 'Chunk at index 1 is missing a title');

    expect($epic->children()->count())->toBe(0);
});

it('refuses a parent in another workspace', function () {
    $other = Workspace::factory()->create();
    $theirs = CreateIssue::run(['title' => 'Theirs', 'type' => Issue::TYPE_EPIC], $other->id);

    CreateIssue::run(['title' => 'Mine', 'parent_id' => $theirs->id], $this->workspace->id);
})->throws(InvalidArgumentException::class, 'belongs to a different workspace');

// ─── Closing ─────────────────────────────────────────────────────────────

it('will not close an epic while a child is open', function () {
    $epic = CreateIssue::run(['title' => 'Long haul', 'type' => Issue::TYPE_EPIC], $this->workspace->id);
    $children = SplitIssue::run($epic, [['title' => 'One'], ['title' => 'Two']]);

    $children->first()->close();

    expect(fn () => $epic->close())
        ->toThrow(RuntimeException::class, '1 child issue(s) still open');

    expect($epic->fresh()->status)->toBe(Issue::STATUS_OPEN);
});

it('closes an epic once every child has closed', function () {
    $epic = CreateIssue::run(['title' => 'Long haul', 'type' => Issue::TYPE_EPIC], $this->workspace->id);
    $children = SplitIssue::run($epic, [['title' => 'One'], ['title' => 'Two']]);

    $children->each(fn (Issue $child) => $child->close());

    expect($epic->getProgress())->toMatchArray(['total' => 2, 'closed' => 2, 'open' => 0, 'percent' => 100]);

    $epic->close();

    expect($epic->fresh()->status)->toBe(Issue::STATUS_CLOSED)
        ->and($epic->fresh()->closed_at)->not->toBeNull();
});

it('closes an epic with open children when forced', function () {
    $epic = CreateIssue::run(['title' => 'Abandoned', 'type' => Issue::TYPE_EPIC], $this->workspace->id);
    SplitIssue::run($epic, [['title' => 'Never done']]);

    $epic->close(force: true);

    expect($epic->fresh()->status)->toBe(Issue::STATUS_CLOSED);
});

it('holds a status update to the same rule as close', function () {
    $epic = CreateIssue::run(['title' => 'Via update', 'type' => Issue::TYPE_EPIC], $this->workspace->id);
    SplitIssue::run($epic, [['title' => 'Outstanding']]);

    UpdateIssue::run($epic->slug, ['status' => Issue::STATUS_CLOSED], $this->workspace->id);
})->throws(RuntimeException::class, 'child issue(s) still open');

it('closes a childless issue without complaint', function () {
    $task = CreateIssue::run(['title' => 'Standalone'], $this->workspace->id);

    expect($task->getProgress())->toMatchArray(['total' => 0, 'percent' => 100]);

    $task->close();

    expect($task->fresh()->status)->toBe(Issue::STATUS_CLOSED);
});

// ─── Discipline and repository ───────────────────────────────────────────

it('rejects an unknown discipline', function () {
    CreateIssue::run(['title' => 'Bad', 'discipline' => 'vibes'], $this->workspace->id);
})->throws(InvalidArgumentException::class, 'discipline must be one of');

it('filters a workspace by discipline and repository', function () {
    $epic = CreateIssue::run([
        'title' => 'Fleet sweep',
        'type' => Issue::TYPE_EPIC,
        'repository' => 'dappcore/go-html',
    ], $this->workspace->id);

    SplitIssue::run($epic, [
        ['title' => 'Write it', 'discipline' => Issue::DISCIPLINE_CODE],
        ['title' => 'Review it', 'discipline' => Issue::DISCIPLINE_REVIEW],
        ['title' => 'Document it', 'discipline' => Issue::DISCIPLINE_CONTENT, 'repository' => 'dappcore/docs'],
    ]);

    expect(Issue::ofDiscipline(Issue::DISCIPLINE_CODE)->count())->toBe(1)
        ->and(Issue::forRepository('dappcore/go-html')->count())->toBe(3)
        ->and(Issue::forRepository('dappcore/docs')->count())->toBe(1)
        ->and(Issue::epics()->count())->toBe(1)
        ->and(Issue::topLevel()->count())->toBe(1)
        ->and(Issue::childrenOf($epic->id)->count())->toBe(3);
});
