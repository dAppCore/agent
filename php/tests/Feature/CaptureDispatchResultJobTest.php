<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\CaptureDispatchResultJob;
use Core\Mod\Agentic\Services\MantisClient;
use Core\Mod\Agentic\Services\ShaExtractor;
use Illuminate\Config\Repository;
use Illuminate\Container\Container;
use Illuminate\Http\Client\Factory;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Facade;
use Illuminate\Support\Facades\Http;

beforeEach(function (): void {
    $container = new Container;
    $config = new Repository([
        'agentic' => [
            'mantis' => [
                'base_url' => 'https://tasks.example.test',
                'token' => 'mantis-token-123',
            ],
        ],
    ]);
    $http = new Factory;

    $container->instance('config', $config);
    $container->instance(Repository::class, $config);
    $container->instance(Factory::class, $http);

    Container::setInstance($container);
    Facade::clearResolvedInstances();
    Facade::setFacadeApplication($container);
    Http::swap($http);
});

test('ShaExtractor_extract_Good_returns_repo_and_url_for_forge_commit_links', function (): void {
    $result = (new ShaExtractor)->extract(
        "Applied fix in https://forge.lthn.sh/core/agent/commit/abcdef1234567 and queued verification.",
    );

    expect($result)->toBe([
        'sha' => 'abcdef1234567',
        'repo' => 'core/agent',
        'forge_url' => 'https://forge.lthn.sh/core/agent/commit/abcdef1234567',
    ]);
});

test('ShaExtractor_extract_Good_returns_first_bare_sha_when_no_forge_link_exists', function (): void {
    $result = (new ShaExtractor)->extract(
        "Summary line\nCommit applied at abcdef1 with follow-up on fedcba9876543.",
    );

    expect($result)->toBe([
        'sha' => 'abcdef1',
        'repo' => null,
        'forge_url' => null,
    ]);
});

test('ShaExtractor_extract_Bad_returns_nulls_when_model_output_has_no_sha', function (): void {
    $result = (new ShaExtractor)->extract('Summary only with no commit reference.');

    expect($result)->toBe([
        'sha' => null,
        'repo' => null,
        'forge_url' => null,
    ]);
});

test('MantisClient_note_Good_posts_issue_note_to_mantis', function (): void {
    Http::fake([
        'https://tasks.example.test/api/rest/issues/828/notes' => Http::response([], 201),
    ]);

    (new MantisClient)->note(828, 'Dispatch completed cleanly.');

    Http::assertSent(fn (Request $request): bool => $request->method() === 'POST'
        && $request->url() === 'https://tasks.example.test/api/rest/issues/828/notes'
        && $request->hasHeader('Authorization', 'mantis-token-123')
        && $request['text'] === 'Dispatch completed cleanly.');
});

test('MantisClient_close_Good_patches_closed_fixed_resolution', function (): void {
    Http::fake([
        'https://tasks.example.test/api/rest/issues/828' => Http::response([], 200),
    ]);

    (new MantisClient)->close(828);

    Http::assertSent(fn (Request $request): bool => $request->method() === 'PATCH'
        && $request->url() === 'https://tasks.example.test/api/rest/issues/828'
        && $request->hasHeader('Authorization', 'mantis-token-123')
        && $request['status']['name'] === 'closed'
        && $request['resolution']['name'] === 'fixed');
});

test('CaptureDispatchResultJob_handle_Good_writes_close_note_then_closes_ticket_from_forge_link', function (): void {
    $requests = [];

    Http::fake(function (Request $request) use (&$requests) {
        $requests[] = [
            'method' => $request->method(),
            'url' => $request->url(),
            'data' => $request->data(),
        ];

        return str_ends_with($request->url(), '/notes')
            ? Http::response([], 201)
            : Http::response([], 200);
    });

    $job = new CaptureDispatchResultJob(
        ticketId: 828,
        response: [
            'output' => [
                'summary' => "Implemented queueable dispatch capture.\nAdditional detail below.",
                'content' => "Implemented queueable dispatch capture.\nCommit: https://forge.lthn.sh/core/agent/commit/abcdef1234567",
            ],
        ],
    );

    $job->handle(new MantisClient, new ShaExtractor);

    expect($requests)->toHaveCount(2);
    expect($requests[0]['method'])->toBe('POST');
    expect($requests[0]['url'])->toBe('https://tasks.example.test/api/rest/issues/828/notes');
    expect($requests[0]['data']['text'])->toBe(
        "Implemented by codex exec supervised by CorePHP Agentic dispatch (profile: unknown).\n"
        ."Commit: abcdef1234567 on core/agent dev (https://forge.lthn.sh/core/agent/commit/abcdef1234567).\n"
        ."Implemented queueable dispatch capture.\n\n"
        .'Filed-by: agentic'
    );
    expect($requests[1]['method'])->toBe('PATCH');
    expect($requests[1]['url'])->toBe('https://tasks.example.test/api/rest/issues/828');
    expect($requests[1]['data'])->toBe([
        'status' => ['name' => 'closed'],
        'resolution' => ['name' => 'fixed'],
    ]);
});

test('CaptureDispatchResultJob_handle_Ugly_builds_forge_url_from_bare_sha_and_repo_hint', function (): void {
    $requests = [];

    Http::fake(function (Request $request) use (&$requests) {
        $requests[] = [
            'method' => $request->method(),
            'url' => $request->url(),
            'data' => $request->data(),
        ];

        return str_ends_with($request->url(), '/notes')
            ? Http::response([], 201)
            : Http::response([], 200);
    });

    $job = new CaptureDispatchResultJob(
        ticketId: 828,
        response: [
            'events' => [
                ['message' => 'Bare SHA emitted from event stream'],
                ['content' => "Commit landed as fedcba9\nVerification pending."],
            ],
            'output' => [
                'summary' => "Bare SHA emitted from event stream\nSecond line ignored.",
            ],
        ],
        repo: 'core/agent',
    );

    $job->handle(new MantisClient, new ShaExtractor);

    expect($requests[0]['data']['text'])->toContain('Commit: fedcba9 on core/agent dev (https://forge.lthn.sh/core/agent/commit/fedcba9).');
    expect($requests[0]['data']['text'])->toContain("Bare SHA emitted from event stream\n\nFiled-by: agentic");
});

test('CaptureDispatchResultJob_handle_Bad_throws_when_no_sha_can_be_extracted', function (): void {
    Http::fake();

    $job = new CaptureDispatchResultJob(
        ticketId: 828,
        response: [
            'output' => [
                'summary' => 'Summary without a commit reference.',
            ],
        ],
        repo: 'core/agent',
    );

    expect(fn () => $job->handle(new MantisClient, new ShaExtractor))
        ->toThrow(RuntimeException::class, 'Unable to extract commit SHA from model output.');

    Http::assertNothingSent();
});
