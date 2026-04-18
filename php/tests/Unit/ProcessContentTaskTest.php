<?php

declare(strict_types=1);

/**
 * Tests for the ProcessContentTask queued job.
 *
 * Covers the handle() flow: prompt validation, entitlement checks,
 * provider availability, task completion, usage recording, and
 * template variable interpolation.
 */

use Core\Mod\Agentic\Jobs\ProcessContentTask;
use Core\Mod\Agentic\Services\AgenticManager;
use Core\Mod\Agentic\Services\AgenticProviderInterface;
use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Tenant\Services\EntitlementService;
use Mod\Content\Models\ContentTask;

// =========================================================================
// Helpers
// =========================================================================

/**
 * Build a minimal mock ContentTask with sensible defaults.
 *
 * @param  array<string,mixed>  $attributes  Override property returns.
 */
function makeTask(array $attributes = []): ContentTask
{
    $task = Mockery::mock(ContentTask::class);
    $task->shouldReceive('markProcessing')->byDefault();
    $task->shouldReceive('markCompleted')->byDefault();
    $task->shouldReceive('markFailed')->byDefault();

    // Set properties directly on the mock — Mockery handles __get/__set
    // internally. Using shouldReceive('__get') doesn't reliably intercept
    // property access on mocks of non-existent classes.
    foreach ($attributes as $key => $value) {
        $task->{$key} = $value;
    }

    return $task;
}

/**
 * Build a minimal mock prompt object.
 */
function makePrompt(string $model = 'claude-sonnet-4-20250514', string $userTemplate = 'Hello'): object
{
    return (object) [
        'id' => 1,
        'model' => $model,
        'system_prompt' => 'You are a helpful assistant.',
        'user_template' => $userTemplate,
        'model_config' => [],
    ];
}

/**
 * Build an AgenticResponse suitable for testing.
 */
function makeResponse(string $content = 'Generated content'): AgenticResponse
{
    return new AgenticResponse(
        content: $content,
        model: 'claude-sonnet-4-20250514',
        inputTokens: 100,
        outputTokens: 50,
        durationMs: 1200,
        stopReason: 'end_turn',
    );
}

afterEach(function () {
    Mockery::close();
});

// =========================================================================
// handle() — prompt validation
// =========================================================================

describe('handle — prompt validation', function () {
    it('marks task as failed when prompt is missing', function () {
        $task = makeTask(['prompt' => null]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')->once()->with('Prompt not found');

        $job = new ProcessContentTask($task);
        $job->handle(
            Mockery::mock(AgenticManager::class),
            Mockery::mock(EntitlementService::class),
        );
    });

    it('does not call AI provider when prompt is missing', function () {
        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldNotReceive('provider');

        $task = makeTask(['prompt' => null]);

        $job = new ProcessContentTask($task);
        $job->handle($ai, Mockery::mock(EntitlementService::class));
    });
});

// =========================================================================
// handle() — entitlement checks
// =========================================================================

describe('handle — entitlement checks', function () {
    it('marks task as failed when entitlement is denied', function () {
        $workspace = Mockery::mock('Core\Tenant\Models\Workspace');
        $entitlementResult = Mockery::mock();
        $entitlementResult->shouldReceive('isDenied')->andReturn(true);
        $entitlementResult->message = 'No AI credits remaining';

        $task = makeTask([
            'prompt' => makePrompt(),
            'workspace' => $workspace,
        ]);
        $task->shouldReceive('markFailed')
            ->once()
            ->with(Mockery::on(fn ($msg) => str_contains($msg, 'Entitlement denied')));

        $entitlements = Mockery::mock(EntitlementService::class);
        $entitlements->shouldReceive('can')
            ->with($workspace, 'ai.credits')
            ->andReturn($entitlementResult);

        $job = new ProcessContentTask($task);
        $job->handle(Mockery::mock(AgenticManager::class), $entitlements);
    });

    it('skips entitlement check when task has no workspace', function () {
        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(false);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $task = makeTask([
            'prompt' => makePrompt(),
            'workspace' => null,
        ]);

        $entitlements = Mockery::mock(EntitlementService::class);
        $entitlements->shouldNotReceive('can');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// handle() — provider availability
// =========================================================================

describe('handle — provider availability', function () {
    it('marks task as failed when provider is unavailable', function () {
        $prompt = makePrompt('gemini-2.0-flash');

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(false);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->with('gemini-2.0-flash')->andReturn($provider);

        $task = makeTask([
            'prompt' => $prompt,
            'workspace' => null,
        ]);
        $task->shouldReceive('markFailed')
            ->once()
            ->with("AI provider 'gemini-2.0-flash' is not configured");

        $job = new ProcessContentTask($task);
        $job->handle($ai, Mockery::mock(EntitlementService::class));
    });
});

// =========================================================================
// handle() — successful completion
// =========================================================================

describe('handle — successful completion', function () {
    it('marks task as completed with response metadata', function () {
        $prompt = makePrompt('claude-sonnet-4-20250514', 'Write about PHP.');
        $response = makeResponse('PHP is a great language.');

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->with('claude-sonnet-4-20250514')->andReturn($provider);

        $task = makeTask([
            'prompt' => $prompt,
            'workspace' => null,
            'input_data' => [],
        ]);
        $task->shouldReceive('markCompleted')
            ->once()
            ->with('PHP is a great language.', Mockery::on(function ($metadata) {
                return $metadata['tokens_input'] === 100
                    && $metadata['tokens_output'] === 50
                    && $metadata['model'] === 'claude-sonnet-4-20250514'
                    && $metadata['duration_ms'] === 1200
                    && isset($metadata['estimated_cost']);
            }));

        $job = new ProcessContentTask($task);
        $job->handle($ai, Mockery::mock(EntitlementService::class));
    });

    it('does not record usage when task has no workspace', function () {
        $response = makeResponse();
        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $task = makeTask([
            'prompt' => makePrompt(),
            'workspace' => null,
            'input_data' => [],
        ]);

        $entitlements = Mockery::mock(EntitlementService::class);
        $entitlements->shouldNotReceive('recordUsage');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('records AI usage when workspace is present', function () {
        $workspace = Mockery::mock('Core\Tenant\Models\Workspace');
        $entitlementResult = Mockery::mock();
        $entitlementResult->shouldReceive('isDenied')->andReturn(false);

        $response = makeResponse();
        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $prompt = makePrompt();
        $task = makeTask([
            'prompt' => $prompt,
            'workspace' => $workspace,
            'input_data' => [],
            'id' => 42,
        ]);

        $entitlements = Mockery::mock(EntitlementService::class);
        $entitlements->shouldReceive('can')
            ->with($workspace, 'ai.credits')
            ->andReturn($entitlementResult);
        $entitlements->shouldReceive('recordUsage')
            ->once()
            ->withSomeOfArgs($workspace, 'ai.credits');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// handle() — template variable interpolation
// =========================================================================

describe('handle — template variable interpolation', function () {
    it('replaces string placeholders in user template', function () {
        $prompt = makePrompt('claude-sonnet-4-20250514', 'Write about {{{topic}}}.');
        $response = makeResponse();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(
                $prompt->system_prompt,
                'Write about Laravel.',
                Mockery::any(),
            )
            ->once()
            ->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $task = makeTask([
            'prompt' => $prompt,
            'workspace' => null,
            'input_data' => ['topic' => 'Laravel'],
        ]);

        $job = new ProcessContentTask($task);
        $job->handle($ai, Mockery::mock(EntitlementService::class));
    });

    it('JSON-encodes array values in template', function () {
        $prompt = makePrompt('claude-sonnet-4-20250514', 'Tags: {{{tags}}}.');
        $response = makeResponse();
        $tags = ['php', 'laravel'];

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(
                $prompt->system_prompt,
                'Tags: '.json_encode($tags).'.',
                Mockery::any(),
            )
            ->once()
            ->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $task = makeTask([
            'prompt' => $prompt,
            'workspace' => null,
            'input_data' => ['tags' => $tags],
        ]);

        $job = new ProcessContentTask($task);
        $job->handle($ai, Mockery::mock(EntitlementService::class));
    });

    it('leaves unknown placeholders untouched', function () {
        $prompt = makePrompt('claude-sonnet-4-20250514', 'Hello {{{name}}}, see {{{unknown}}}.');
        $response = makeResponse();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(
                $prompt->system_prompt,
                'Hello World, see {{{unknown}}}.',
                Mockery::any(),
            )
            ->once()
            ->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $task = makeTask([
            'prompt' => $prompt,
            'workspace' => null,
            'input_data' => ['name' => 'World'],
        ]);

        $job = new ProcessContentTask($task);
        $job->handle($ai, Mockery::mock(EntitlementService::class));
    });
});

// =========================================================================
// failed() — queue failure handler
// =========================================================================

describe('failed', function () {
    it('marks task as failed with exception message', function () {
        $exception = new RuntimeException('Connection refused');

        $task = makeTask();
        $task->shouldReceive('markFailed')->once()->with('Connection refused');

        $job = new ProcessContentTask($task);
        $job->failed($exception);
    });
});
