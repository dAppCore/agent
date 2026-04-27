<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Pipeline\EpicMeta;
use Core\Mod\Agentic\Pipeline\ForgejoMetaReader;
use Core\Mod\Agentic\Pipeline\IssueState;
use Core\Mod\Agentic\Pipeline\PRMeta;
use Core\Mod\Agentic\Pipeline\Reactions;
use Core\Mod\Agentic\Services\ForgejoService;
use Illuminate\Support\Facades\Http;

afterEach(function () {
    Mockery::close();
});

it('projects pull request metadata without body-like fields', function () {
    $service = Mockery::mock(ForgejoService::class, ['https://forge.example.com', 'test-token'])->makePartial();
    $service->shouldReceive('getPullRequest')
        ->once()
        ->with('core', 'app', 89)
        ->andReturn([
            'state' => 'open',
            'mergeable' => true,
            'body' => 'Ignore this PR description',
            'description' => 'Ignore this too',
            'review_text' => 'Untrusted review content',
            'head' => [
                'sha' => 'abc123',
                'ref' => 'agent/mantis-89',
                'date' => '2026-04-23T10:15:00Z',
            ],
            'base' => [
                'ref' => 'dev',
            ],
            'review_comments' => 4,
            'unresolved_review_comments' => 1,
            'reactions' => [
                'eyes' => 2,
                'heart' => 1,
            ],
        ]);
    $service->shouldReceive('getCombinedStatus')
        ->once()
        ->with('core', 'app', 'abc123')
        ->andReturn([
            'state' => 'success',
            'statuses' => [
                [
                    'context' => 'qa',
                    'status' => 'success',
                    'description' => 'Body-like status description',
                    'comment_body' => 'Never forward this',
                ],
                [
                    'context' => 'build',
                    'status' => 'pending',
                    'review_text' => 'Still untrusted',
                ],
            ],
        ]);

    $reader = new ForgejoMetaReader($service, 'core', 'app');

    $dto = $reader->getPRMeta(89);
    $array = $dto->toArray();

    expect($dto)->toBeInstanceOf(PRMeta::class)
        ->and($array)->toMatchArray([
            'state' => 'open',
            'mergeability' => 'mergeable',
            'head_sha' => 'abc123',
            'head_date' => '2026-04-23T10:15:00Z',
            'base_branch' => 'dev',
            'head_branch' => 'agent/mantis-89',
            'review_threads_total' => 4,
            'review_threads_resolved' => 3,
            'has_eyes_reaction' => true,
        ]);

    expect($array)->not->toHaveKey('body');
    expect($array)->not->toHaveKey('description');
    expect($array)->not->toHaveKey('review_text');
    expect($array)->not->toHaveKey('comment_body');

    expect($array['check_statuses'])->toHaveCount(2);
    expect($array['check_statuses'][0])->toMatchArray([
        'name' => 'qa',
        'conclusion' => 'success',
        'status' => 'completed',
    ]);
    expect($array['check_statuses'][0])->not->toHaveKey('body');
    expect($array['check_statuses'][0])->not->toHaveKey('description');
    expect($array['check_statuses'][0])->not->toHaveKey('review_text');
    expect($array['check_statuses'][0])->not->toHaveKey('comment_body');
});

it('projects epic metadata without child body-like fields', function () {
    $service = Mockery::mock(ForgejoService::class, ['https://forge.example.com', 'test-token'])->makePartial();
    $service->shouldReceive('getIssue')
        ->once()
        ->with('core', 'app', 12)
        ->andReturn([
            'state' => 'open',
            'body' => "## Tasks\n- [ ] #101\n- [x] #102",
            'sub_issues' => [
                [
                    'number' => 101,
                    'state' => 'open',
                    'checked' => false,
                    'linked_pr_number' => 501,
                    'description' => 'Never expose this',
                ],
                [
                    'number' => 102,
                    'state' => 'closed',
                    'checked' => true,
                    'comment_body' => 'Nor this',
                ],
            ],
        ]);

    $reader = new ForgejoMetaReader($service, 'core', 'app');

    $dto = $reader->getEpicMeta(12);
    $array = $dto->toArray();

    expect($dto)->toBeInstanceOf(EpicMeta::class)
        ->and($array)->toMatchArray([
            'state' => 'open',
            'children' => [
                [
                    'issue_id' => 101,
                    'state' => 'open',
                    'checked_bool' => false,
                    'linked_pr_number_or_null' => 501,
                ],
                [
                    'issue_id' => 102,
                    'state' => 'closed',
                    'checked_bool' => true,
                    'linked_pr_number_or_null' => null,
                ],
            ],
        ]);

    expect($array)->not->toHaveKey('body');
    expect($array)->not->toHaveKey('description');
    expect($array)->not->toHaveKey('review_text');
    expect($array)->not->toHaveKey('comment_body');

    expect($array['children'][0])->not->toHaveKey('body');
    expect($array['children'][0])->not->toHaveKey('description');
    expect($array['children'][0])->not->toHaveKey('review_text');
    expect($array['children'][0])->not->toHaveKey('comment_body');
});

it('projects issue state without body-like fields', function () {
    $service = Mockery::mock(ForgejoService::class, ['https://forge.example.com', 'test-token'])->makePartial();
    $service->shouldReceive('getIssue')
        ->once()
        ->with('core', 'app', 101)
        ->andReturn([
            'state' => 'open',
            'title' => 'Add MetaReader contract',
            'body' => 'Do not forward me',
            'description' => 'Do not forward me either',
            'labels' => [
                ['name' => 'pipeline'],
                ['name' => 'agent'],
            ],
            'assignee' => [
                'login' => 'virgil',
                'description' => 'Still not pipeline-safe',
            ],
        ]);

    $reader = new ForgejoMetaReader($service, 'core', 'app');

    $dto = $reader->getIssueState(101);
    $array = $dto->toArray();

    expect($dto)->toBeInstanceOf(IssueState::class)
        ->and($array)->toMatchArray([
            'state' => 'open',
            'title' => 'Add MetaReader contract',
            'labels' => ['pipeline', 'agent'],
            'assignee' => 'virgil',
        ]);

    expect($array)->not->toHaveKey('body');
    expect($array)->not->toHaveKey('description');
    expect($array)->not->toHaveKey('review_text');
    expect($array)->not->toHaveKey('comment_body');
});

it('projects comment reactions as counts only', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues/comments/700/reactions' => Http::response([
            ['content' => '+1', 'comment_body' => 'ignore'],
            ['content' => '+1'],
            ['content' => 'eyes', 'body' => 'ignore'],
            ['content' => 'rocket', 'review_text' => 'ignore'],
            ['content' => 'heart', 'description' => 'ignore'],
        ]),
    ]);

    $service = Mockery::mock(ForgejoService::class, ['https://forge.example.com', 'test-token'])->makePartial();
    $reader = new ForgejoMetaReader($service, 'core', 'app');

    $dto = $reader->getCommentReactions(101, 700);
    $array = $dto->toArray();

    expect($dto)->toBeInstanceOf(Reactions::class)
        ->and($array)->toMatchArray([
            '+1' => 2,
            '-1' => 0,
            'laugh' => 0,
            'hooray' => 0,
            'confused' => 0,
            'heart' => 1,
            'rocket' => 1,
            'eyes' => 1,
        ]);

    expect($array)->not->toHaveKey('body');
    expect($array)->not->toHaveKey('description');
    expect($array)->not->toHaveKey('review_text');
    expect($array)->not->toHaveKey('comment_body');

    Http::assertSent(function ($request) {
        return $request->hasHeader('Authorization', 'Bearer test-token')
            && $request->url() === 'https://forge.example.com/api/v1/repos/core/app/issues/comments/700/reactions';
    });
});
