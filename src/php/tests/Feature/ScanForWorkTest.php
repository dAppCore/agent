<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Forge\ScanForWork;
use Core\Mod\Agentic\Services\ForgejoService;
use Illuminate\Support\Facades\Http;

beforeEach(function () {
    $this->service = new ForgejoService(
        baseUrl: 'https://forge.example.com',
        token: 'test-token-abc',
    );

    $this->app->instance(ForgejoService::class, $this->service);
});

it('finds unchecked children needing coding', function () {
    Http::fake([
        // List epic issues
        'forge.example.com/api/v1/repos/core/app/issues?state=open&type=issues&labels=epic' => Http::response([
            [
                'number' => 1,
                'title' => 'Epic: Build the widget',
                'body' => "## Tasks\n- [ ] #2\n- [x] #3\n- [ ] #4",
            ],
        ]),

        // List PRs — only #4 has a linked PR
        'forge.example.com/api/v1/repos/core/app/pulls?state=all' => Http::response([
            [
                'number' => 10,
                'title' => 'Fix for issue 4',
                'body' => 'Closes #4',
            ],
        ]),

        // Child issue #2 (no PR, should be returned)
        'forge.example.com/api/v1/repos/core/app/issues/2' => Http::response([
            'number' => 2,
            'title' => 'Add colour picker',
            'body' => 'We need a colour picker component.',
            'assignees' => [['login' => 'virgil']],
        ]),
    ]);

    $items = ScanForWork::run('core', 'app');

    expect($items)->toBeArray()->toHaveCount(1);
    expect($items[0])->toMatchArray([
        'epic_number' => 1,
        'issue_number' => 2,
        'issue_title' => 'Add colour picker',
        'issue_body' => 'We need a colour picker component.',
        'assignee' => 'virgil',
        'repo_owner' => 'core',
        'repo_name' => 'app',
        'needs_coding' => true,
        'has_pr' => false,
    ]);
});

it('skips checked items and items with PRs', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues?state=open&type=issues&labels=epic' => Http::response([
            [
                'number' => 1,
                'title' => 'Epic: Build the widget',
                'body' => "- [x] #2\n- [x] #3\n- [ ] #4",
            ],
        ]),

        'forge.example.com/api/v1/repos/core/app/pulls?state=all' => Http::response([
            [
                'number' => 10,
                'title' => 'Fix for issue 4',
                'body' => 'Resolves #4',
            ],
        ]),
    ]);

    $items = ScanForWork::run('core', 'app');

    expect($items)->toBeArray()->toHaveCount(0);
});

it('returns empty for repos with no epics', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues?state=open&type=issues&labels=epic' => Http::response([]),
    ]);

    $items = ScanForWork::run('core', 'app');

    expect($items)->toBeArray()->toHaveCount(0);
});
