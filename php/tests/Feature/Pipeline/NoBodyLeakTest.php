<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Forge\ManagePullRequest;
use Core\Mod\Agentic\Actions\Forge\ScanForWork;
use Core\Mod\Agentic\Pipeline\EpicChild;
use Core\Mod\Agentic\Pipeline\EpicMeta;
use Core\Mod\Agentic\Pipeline\IssueState;
use Core\Mod\Agentic\Pipeline\MetaReader;
use Core\Mod\Agentic\Pipeline\PRMeta;
use Core\Mod\Agentic\Services\ForgejoService;

afterEach(function () {
    Mockery::close();
});

function expectNoBodyLikeKeys(mixed $value): void
{
    if (! is_array($value)) {
        return;
    }

    expect($value)->not->toHaveKey('body');
    expect($value)->not->toHaveKey('description');
    expect($value)->not->toHaveKey('review_text');
    expect($value)->not->toHaveKey('comment_body');
    expect($value)->not->toHaveKey('issue_body');

    foreach ($value as $nested) {
        expectNoBodyLikeKeys($nested);
    }
}

it('keeps scan for work output free of body-like fields', function () {
    $forgejo = Mockery::mock(ForgejoService::class);
    $metaReader = Mockery::mock(MetaReader::class);

    $forgejo->shouldReceive('listIssues')
        ->once()
        ->with('core', 'app', 'open', 'epic')
        ->andReturn([
            [
                'number' => 90,
                'body' => "- [ ] #101\n- [x] #102",
                'description' => 'Ignore this epic description',
                'review_text' => 'Ignore this review text',
            ],
        ]);
    $forgejo->shouldNotReceive('listPullRequests');
    $forgejo->shouldNotReceive('getIssue');

    $metaReader->shouldReceive('getEpicMeta')
        ->once()
        ->with(90)
        ->andReturn(new EpicMeta('open', [
            new EpicChild(101, 'open', false, null),
            new EpicChild(102, 'open', true, null),
            new EpicChild(103, 'closed', false, null),
            new EpicChild(104, 'open', false, 700),
        ]));
    $metaReader->shouldReceive('getIssueState')
        ->once()
        ->with(101)
        ->andReturn(new IssueState(
            state: 'open',
            title: 'Add MetaReader scan',
            labels: ['agent', 'pipeline'],
            assignee: 'virgil',
        ));

    $this->app->instance(ForgejoService::class, $forgejo);
    $this->app->instance(MetaReader::class, $metaReader);

    $output = ScanForWork::run('core', 'app');

    expect($output)->toHaveCount(1);
    expect($output[0])->toMatchArray([
        'epic_number' => 90,
        'issue_number' => 101,
        'issue_title' => 'Add MetaReader scan',
        'issue_state' => 'open',
        'issue_labels' => ['agent', 'pipeline'],
        'assignee' => 'virgil',
        'repo_owner' => 'core',
        'repo_name' => 'app',
        'needs_coding' => true,
        'has_pr' => false,
    ]);
    expectNoBodyLikeKeys($output);
});

it('keeps pull request decisions free of body-like fields', function () {
    $forgejo = Mockery::mock(ForgejoService::class);
    $metaReader = Mockery::mock(MetaReader::class);

    $forgejo->shouldNotReceive('getPullRequest');
    $forgejo->shouldNotReceive('getCombinedStatus');
    $forgejo->shouldReceive('mergePullRequest')
        ->once()
        ->with('core', 'app', 77);

    $metaReader->shouldReceive('getPRMeta')
        ->once()
        ->with(77)
        ->andReturn(new PRMeta(
            state: 'open',
            mergeability: 'mergeable',
            headSha: 'abc123',
            headDate: '2026-04-23T12:00:00Z',
            baseBranch: 'dev',
            headBranch: 'agent/mantis-90',
            checkStatuses: [
                [
                    'name' => 'qa',
                    'conclusion' => 'success',
                    'status' => 'completed',
                    'body' => 'Ignore this status body',
                ],
                [
                    'name' => 'review',
                    'conclusion' => 'success',
                    'status' => 'completed',
                    'description' => 'Ignore this status description',
                    'review_text' => 'Ignore this review text',
                ],
            ],
            reviewThreadsTotal: 1,
            reviewThreadsResolved: 1,
            hasEyesReaction: true,
        ));

    $this->app->instance(ForgejoService::class, $forgejo);
    $this->app->instance(MetaReader::class, $metaReader);

    $output = ManagePullRequest::run('core', 'app', 77);

    expect($output)->toMatchArray([
        'merged' => true,
        'pr_number' => 77,
    ]);
    expectNoBodyLikeKeys($output);
});
