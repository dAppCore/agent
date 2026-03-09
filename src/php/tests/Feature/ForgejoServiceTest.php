<?php

use Core\Mod\Agentic\Services\ForgejoService;
use Illuminate\Support\Facades\Http;

beforeEach(function () {
    $this->service = new ForgejoService(
        baseUrl: 'https://forge.example.com',
        token: 'test-token-abc',
    );
});

it('sends bearer token on every request', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues*' => Http::response([]),
    ]);

    $this->service->listIssues('core', 'app');

    Http::assertSent(function ($request) {
        return $request->hasHeader('Authorization', 'Bearer test-token-abc');
    });
});

it('fetches open issues', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues*' => Http::response([
            ['id' => 1, 'number' => 1, 'title' => 'Fix the widget'],
            ['id' => 2, 'number' => 2, 'title' => 'Add colour picker'],
        ]),
    ]);

    $issues = $this->service->listIssues('core', 'app');

    expect($issues)->toBeArray()->toHaveCount(2);
    expect($issues[0]['title'])->toBe('Fix the widget');

    Http::assertSent(function ($request) {
        return str_contains($request->url(), 'state=open')
            && str_contains($request->url(), '/repos/core/app/issues');
    });
});

it('fetches issues filtered by label', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues*' => Http::response([
            ['id' => 3, 'number' => 3, 'title' => 'Labelled issue'],
        ]),
    ]);

    $this->service->listIssues('core', 'app', 'open', 'bug');

    Http::assertSent(function ($request) {
        return str_contains($request->url(), 'labels=bug');
    });
});

it('creates an issue comment', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues/5/comments' => Http::response([
            'id' => 42,
            'body' => 'Agent analysis complete.',
        ], 201),
    ]);

    $comment = $this->service->createComment('core', 'app', 5, 'Agent analysis complete.');

    expect($comment)->toBeArray();
    expect($comment['body'])->toBe('Agent analysis complete.');

    Http::assertSent(function ($request) {
        return $request->method() === 'POST'
            && str_contains($request->url(), '/issues/5/comments')
            && $request['body'] === 'Agent analysis complete.';
    });
});

it('lists pull requests', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/pulls*' => Http::response([
            ['id' => 10, 'number' => 10, 'title' => 'Feature branch'],
        ]),
    ]);

    $prs = $this->service->listPullRequests('core', 'app', 'open');

    expect($prs)->toBeArray()->toHaveCount(1);
    expect($prs[0]['title'])->toBe('Feature branch');

    Http::assertSent(function ($request) {
        return str_contains($request->url(), 'state=open')
            && str_contains($request->url(), '/repos/core/app/pulls');
    });
});

it('gets combined commit status', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/commits/abc123/status' => Http::response([
            'state' => 'success',
            'statuses' => [
                ['context' => 'ci/tests', 'status' => 'success'],
            ],
        ]),
    ]);

    $status = $this->service->getCombinedStatus('core', 'app', 'abc123');

    expect($status['state'])->toBe('success');
    expect($status['statuses'])->toHaveCount(1);
});

it('merges a pull request', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/pulls/7/merge' => Http::response(null, 200),
    ]);

    $this->service->mergePullRequest('core', 'app', 7, 'squash');

    Http::assertSent(function ($request) {
        return $request->method() === 'POST'
            && str_contains($request->url(), '/pulls/7/merge')
            && $request['Do'] === 'squash';
    });
});

it('throws on failed merge', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/pulls/7/merge' => Http::response(
            ['message' => 'not mergeable'],
            405,
        ),
    ]);

    $this->service->mergePullRequest('core', 'app', 7);
})->throws(RuntimeException::class);

it('creates a branch', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/branches' => Http::response([
            'name' => 'agent/fix-123',
        ], 201),
    ]);

    $branch = $this->service->createBranch('core', 'app', 'agent/fix-123', 'main');

    expect($branch['name'])->toBe('agent/fix-123');

    Http::assertSent(function ($request) {
        return $request->method() === 'POST'
            && $request['new_branch_name'] === 'agent/fix-123'
            && $request['old_branch_name'] === 'main';
    });
});

it('adds labels to an issue', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues/3/labels' => Http::response([
            ['id' => 1, 'name' => 'bug'],
            ['id' => 2, 'name' => 'priority'],
        ]),
    ]);

    $labels = $this->service->addLabels('core', 'app', 3, [1, 2]);

    expect($labels)->toBeArray()->toHaveCount(2);

    Http::assertSent(function ($request) {
        return $request->method() === 'POST'
            && $request['labels'] === [1, 2];
    });
});
