<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Forge\AssignAgent;
use Core\Mod\Agentic\Actions\Forge\ManagePullRequest;
use Core\Mod\Agentic\Actions\Forge\ReportToIssue;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Services\ForgejoService;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\Http;

beforeEach(function () {
    $this->workspace = Workspace::factory()->create();

    $this->service = new ForgejoService(
        baseUrl: 'https://forge.example.com',
        token: 'test-token-abc',
    );

    $this->app->instance(ForgejoService::class, $this->service);
});

it('assigns an agent to a plan and starts a session', function () {
    $plan = AgentPlan::factory()->create([
        'workspace_id' => $this->workspace->id,
        'status' => AgentPlan::STATUS_DRAFT,
    ]);

    $session = AssignAgent::run($plan, 'opus', $this->workspace->id);

    expect($session)->toBeInstanceOf(AgentSession::class);
    expect($session->agent_type)->toBe('opus');
    expect($session->agent_plan_id)->toBe($plan->id);

    // Plan should be activated
    $plan->refresh();
    expect($plan->status)->toBe(AgentPlan::STATUS_ACTIVE);
});

it('reports progress to a Forgejo issue', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/issues/5/comments' => Http::response([
            'id' => 1,
            'body' => 'Progress update: phase 1 complete.',
        ]),
    ]);

    ReportToIssue::run('core', 'app', 5, 'Progress update: phase 1 complete.');

    Http::assertSent(function ($request) {
        return str_contains($request->url(), '/repos/core/app/issues/5/comments')
            && $request['body'] === 'Progress update: phase 1 complete.';
    });
});

it('merges a PR when checks pass', function () {
    Http::fake([
        // Get PR — open and mergeable
        'forge.example.com/api/v1/repos/core/app/pulls/10' => Http::response([
            'number' => 10,
            'state' => 'open',
            'mergeable' => true,
            'head' => ['sha' => 'abc123'],
        ]),

        // Combined status — success
        'forge.example.com/api/v1/repos/core/app/commits/abc123/status' => Http::response([
            'state' => 'success',
        ]),

        // Merge
        'forge.example.com/api/v1/repos/core/app/pulls/10/merge' => Http::response([], 200),
    ]);

    $result = ManagePullRequest::run('core', 'app', 10);

    expect($result)->toMatchArray([
        'merged' => true,
        'pr_number' => 10,
    ]);
});

it('does not merge PR when checks are pending', function () {
    Http::fake([
        'forge.example.com/api/v1/repos/core/app/pulls/10' => Http::response([
            'number' => 10,
            'state' => 'open',
            'mergeable' => true,
            'head' => ['sha' => 'abc123'],
        ]),

        'forge.example.com/api/v1/repos/core/app/commits/abc123/status' => Http::response([
            'state' => 'pending',
        ]),
    ]);

    $result = ManagePullRequest::run('core', 'app', 10);

    expect($result)->toMatchArray([
        'merged' => false,
        'reason' => 'checks_pending',
    ]);
});
