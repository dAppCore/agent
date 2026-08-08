<?php

declare(strict_types=1);

/**
 * Tests for the ProcessContentTask queue job.
 *
 * Covers job configuration, execution paths, error handling, retry logic,
 * and the stub processOutput() implementation.
 * Uses Mockery to isolate the job from external dependencies.
 */

use Core\Mod\Agentic\Jobs\ProcessContentTask;
use Core\Mod\Agentic\Models\Prompt;
use Core\Mod\Agentic\Services\AgenticManager;
use Core\Mod\Agentic\Services\AgenticProviderInterface;
use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Mod\Content\Models\ContentTask;
use Core\Tenant\Models\UsageRecord;
use Core\Tenant\Services\EntitlementResult;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Support\Facades\Queue;
use Mockery\MockInterface;

// =========================================================================
// Helpers
// =========================================================================

/**
 * Build a mock ContentTask with sensible defaults.
 *
 * @param  array<string, mixed>  $overrides
 */
function mockContentTask(array $overrides = []): MockInterface
{
    $prompt = Mockery::mock(Prompt::class)->makePartial();
    $prompt->model = $overrides['prompt_model'] ?? 'claude';
    $prompt->user_template = $overrides['user_template'] ?? 'Hello {{name}}';
    $prompt->system_prompt = $overrides['system_prompt'] ?? 'You are helpful.';
    $prompt->model_config = $overrides['model_config'] ?? [];
    $prompt->id = $overrides['prompt_id'] ?? 1;

    $task = Mockery::mock(ContentTask::class)->makePartial();
    $task->id = $overrides['task_id'] ?? 1;
    $task->prompt = array_key_exists('prompt', $overrides) ? $overrides['prompt'] : $prompt;
    $task->workspace = $overrides['workspace'] ?? null;
    $task->input_data = $overrides['input_data'] ?? [];
    $task->target_type = $overrides['target_type'] ?? null;
    $task->target_id = $overrides['target_id'] ?? null;
    $task->target = $overrides['target'] ?? null;

    $task->shouldReceive('markProcessing')->andReturnNull()->byDefault();
    $task->shouldReceive('markFailed')->andReturnNull()->byDefault();
    $task->shouldReceive('markCompleted')->andReturnNull()->byDefault();

    return $task;
}

/**
 * Build a mock AgenticResponse.
 */
function mockAgenticResponse(array $overrides = []): AgenticResponse
{
    return new AgenticResponse(
        content: $overrides['content'] ?? 'Generated content',
        model: $overrides['model'] ?? 'claude-sonnet-4-20250514',
        inputTokens: $overrides['inputTokens'] ?? 100,
        outputTokens: $overrides['outputTokens'] ?? 50,
        stopReason: $overrides['stopReason'] ?? 'end_turn',
        durationMs: $overrides['durationMs'] ?? 1000,
        raw: $overrides['raw'] ?? [],
    );
}

/**
 * Build an EntitlementResult.
 *
 * The real class, not a look-alike: EntitlementService::can() declares
 * EntitlementResult as its return type, so a stand-in with the same two methods
 * fails the return check the moment the service is mocked against its real
 * signature. The old stand-in also published a public $message, which is what
 * let ProcessContentTask read a property EntitlementResult does not have.
 */
function mockEntitlementResult(bool $denied = false, string $message = ''): EntitlementResult
{
    return $denied
        ? EntitlementResult::denied($message, featureCode: 'ai.credits')
        : EntitlementResult::allowed(featureCode: 'ai.credits');
}

// =========================================================================
// Job Configuration Tests
// =========================================================================

describe('job configuration', function () {
    it('retries up to 3 times', function () {
        $task = mockContentTask();
        $job = new ProcessContentTask($task);

        expect($job->tries)->toBe(3);
    });

    it('backs off for 60 seconds between retries', function () {
        $task = mockContentTask();
        $job = new ProcessContentTask($task);

        expect($job->backoff)->toBe(60);
    });

    it('has a 300 second timeout', function () {
        $task = mockContentTask();
        $job = new ProcessContentTask($task);

        expect($job->timeout)->toBe(300);
    });

    it('dispatches to the ai queue', function () {
        Queue::fake();

        $task = mockContentTask();
        ProcessContentTask::dispatch($task);

        Queue::assertPushedOn('ai', ProcessContentTask::class);
    });

    it('implements ShouldQueue', function () {
        $task = mockContentTask();
        $job = new ProcessContentTask($task);

        expect($job)->toBeInstanceOf(ShouldQueue::class);
    });

    it('stores the task on the job', function () {
        $task = mockContentTask(['task_id' => 42]);
        $job = new ProcessContentTask($task);

        expect($job->task->id)->toBe(42);
    });
});

// =========================================================================
// Failed Handler Tests
// =========================================================================

describe('failed handler', function () {
    it('marks the task as failed with the exception message', function () {
        $task = mockContentTask();
        $task->shouldReceive('markFailed')
            ->once()
            ->with('Something went wrong');

        $job = new ProcessContentTask($task);
        $job->failed(new RuntimeException('Something went wrong'));
    });

    it('marks the task as failed with any throwable message', function () {
        $task = mockContentTask();
        $task->shouldReceive('markFailed')
            ->once()
            ->with('Database connection lost');

        $job = new ProcessContentTask($task);
        $job->failed(new Exception('Database connection lost'));
    });

    it('uses the exception message verbatim', function () {
        $task = mockContentTask();

        $capturedMessage = null;
        $task->shouldReceive('markFailed')
            ->once()
            ->andReturnUsing(function (string $message) use (&$capturedMessage) {
                $capturedMessage = $message;
            });

        $job = new ProcessContentTask($task);
        $job->failed(new RuntimeException('Detailed error: code 503'));

        expect($capturedMessage)->toBe('Detailed error: code 503');
    });
});

// =========================================================================
// Handle – Early Exit: Missing Prompt
// =========================================================================

describe('handle with missing prompt', function () {
    it('marks the task failed when prompt is null', function () {
        $task = mockContentTask(['prompt' => null]);

        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')
            ->once()
            ->with('Prompt not found');

        $ai = Mockery::mock(AgenticManager::class);
        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('does not call the AI provider when prompt is missing', function () {
        $task = mockContentTask(['prompt' => null]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')->once();

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldNotReceive('provider');

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// Handle – Early Exit: Entitlement Denied
// =========================================================================

describe('handle with denied entitlement', function () {
    it('marks the task failed when entitlement is denied', function () {
        $workspace = Mockery::mock('Core\Tenant\Models\Workspace');
        $task = mockContentTask(['workspace' => $workspace]);

        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')
            ->once()
            ->with('Entitlement denied: Insufficient credits');

        $ai = Mockery::mock(AgenticManager::class);

        $result = mockEntitlementResult(denied: true, message: 'Insufficient credits');
        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');
        $entitlements->shouldReceive('can')
            ->once()
            ->with($workspace, 'ai.credits')
            ->andReturn($result);

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('does not invoke the AI provider when entitlement is denied', function () {
        $workspace = Mockery::mock('Core\Tenant\Models\Workspace');
        $task = mockContentTask(['workspace' => $workspace]);

        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')->once();

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldNotReceive('provider');

        $result = mockEntitlementResult(denied: true, message: 'Out of credits');
        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');
        $entitlements->shouldReceive('can')->andReturn($result);

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('skips entitlement check when task has no workspace', function () {
        $task = mockContentTask(['workspace' => null]);
        $task->shouldReceive('markProcessing')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(false);
        $provider->shouldReceive('name')->andReturn('claude')->byDefault();

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');
        $entitlements->shouldNotReceive('can');

        $task->shouldReceive('markFailed')
            ->once()
            ->with(Mockery::pattern('/is not configured/'));

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// Handle – Early Exit: Provider Unavailable
// =========================================================================

describe('handle with unavailable provider', function () {
    it('marks the task failed when provider is not configured', function () {
        $task = mockContentTask();
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')
            ->once()
            ->with("AI provider 'claude' is not configured");

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(false);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->with('claude')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('includes the provider name in the failure message', function () {
        $task = mockContentTask(['prompt_model' => 'gemini']);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markFailed')
            ->once()
            ->with("AI provider 'gemini' is not configured");

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(false);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->with('gemini')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// Handle – Successful Execution (without workspace)
// =========================================================================

describe('handle with successful generation (no workspace)', function () {
    it('marks the task as processing then completed', function () {
        $task = mockContentTask([
            'workspace' => null,
            'input_data' => ['name' => 'World'],
        ]);

        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')
            ->once()
            ->with('Generated content', Mockery::type('array'));

        $response = mockAgenticResponse();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->once()
            ->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->with('claude')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('passes interpolated user prompt to the provider', function () {
        $task = mockContentTask([
            'workspace' => null,
            'user_template' => 'Hello {{name}}, your ID is {{id}}',
            'input_data' => ['name' => 'Alice', 'id' => '42'],
        ]);

        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $response = mockAgenticResponse();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->once()
            ->with(
                Mockery::any(),
                'Hello Alice, your ID is 42',
                Mockery::any(),
            )
            ->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('passes system prompt to the provider', function () {
        $task = mockContentTask([
            'workspace' => null,
            'system_prompt' => 'You are a content writer.',
        ]);

        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $response = mockAgenticResponse();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->once()
            ->with('You are a content writer.', Mockery::any(), Mockery::any())
            ->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('includes token and cost metadata when marking completed', function () {
        $task = mockContentTask(['workspace' => null]);
        $task->shouldReceive('markProcessing')->once();

        $capturedMeta = null;
        $task->shouldReceive('markCompleted')
            ->once()
            ->andReturnUsing(function (string $content, array $meta) use (&$capturedMeta) {
                $capturedMeta = $meta;
            });

        $response = mockAgenticResponse([
            'inputTokens' => 120,
            'outputTokens' => 60,
            'model' => 'claude-sonnet-4-20250514',
            'durationMs' => 2500,
        ]);

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);

        expect($capturedMeta)
            ->toHaveKey('tokens_input', 120)
            ->toHaveKey('tokens_output', 60)
            ->toHaveKey('model', 'claude-sonnet-4-20250514')
            ->toHaveKey('duration_ms', 2500)
            ->toHaveKey('estimated_cost');
    });

    it('does not record usage when workspace is absent', function () {
        $task = mockContentTask(['workspace' => null]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');
        $entitlements->shouldNotReceive('recordUsage');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// Handle – Successful Execution (with workspace)
// =========================================================================

describe('handle with successful generation (with workspace)', function () {
    it('records AI usage after successful generation', function () {
        $workspace = Mockery::mock('Core\Tenant\Models\Workspace');
        $task = mockContentTask(['workspace' => $workspace, 'task_id' => 7]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $response = mockAgenticResponse(['inputTokens' => 80, 'outputTokens' => 40]);

        $allowedResult = mockEntitlementResult(denied: false);

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');
        $entitlements->shouldReceive('can')
            ->once()
            ->with($workspace, 'ai.credits')
            ->andReturn($allowedResult);
        $entitlements->shouldReceive('recordUsage')
            ->once()
            // Positional, matching recordUsage($workspace, $featureCode,
            // $quantity, $user, $metadata): the call arrives with all five slots
            // filled, so a named-argument expectation has no entry for $user and
            // Mockery fails looking up argument 3.
            ->with($workspace, 'ai.credits', 1, null, Mockery::type('array'))
            // recordUsage() returns a non-nullable UsageRecord, so an
            // expectation without andReturn() fails the return type check.
            ->andReturn(Mockery::mock(UsageRecord::class));

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('includes task and prompt metadata in usage recording', function () {
        $workspace = Mockery::mock('Core\Tenant\Models\Workspace');
        $task = mockContentTask([
            'workspace' => $workspace,
            'task_id' => 99,
            'prompt_id' => 5,
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $response = mockAgenticResponse();
        $allowedResult = mockEntitlementResult(denied: false);

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn($response);

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $capturedMeta = null;
        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');
        $entitlements->shouldReceive('can')->andReturn($allowedResult);
        $entitlements->shouldReceive('recordUsage')
            ->once()
            // Five parameters, matching recordUsage(): the job passes quantity
            // and metadata as named arguments and skips $user, so $user still
            // arrives positionally as null between them.
            ->andReturnUsing(function ($ws, $key, $quantity, $user, $metadata) use (&$capturedMeta) {
                $capturedMeta = $metadata;

                return Mockery::mock(UsageRecord::class);
            });

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);

        expect($capturedMeta)
            ->toHaveKey('task_id', 99)
            ->toHaveKey('prompt_id', 5);
    });
});

// =========================================================================
// Handle – processOutput Stub Tests
// =========================================================================

describe('processOutput stub', function () {
    it('completes without error when task has no target', function () {
        $task = mockContentTask([
            'workspace' => null,
            'target_type' => null,
            'target_id' => null,
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);

        // Should complete without exception
        expect(fn () => $job->handle($ai, $entitlements))->not->toThrow(Exception::class);
    });

    it('completes without error when task has a target but no matching model (stub behaviour)', function () {
        // processOutput() is currently a stub: it logs nothing and returns
        // when the target is null. This test documents the stub behaviour.
        $task = mockContentTask([
            'workspace' => null,
            'target_type' => 'App\\Models\\Article',
            'target_id' => 1,
            'target' => null, // target relationship not resolved
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);

        expect(fn () => $job->handle($ai, $entitlements))->not->toThrow(Exception::class);
    });

    it('calls processOutput when both target_type and target_id are set', function () {
        $target = Mockery::mock('stdClass');

        $task = mockContentTask([
            'workspace' => null,
            'target_type' => 'App\\Models\\Article',
            'target_id' => 5,
            'target' => $target,
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);

        expect(fn () => $job->handle($ai, $entitlements))->not->toThrow(Exception::class);
    });
});

// =========================================================================
// Variable Interpolation Tests (via handle())
// =========================================================================

describe('variable interpolation', function () {
    it('replaces single string placeholder', function () {
        $task = mockContentTask([
            'workspace' => null,
            'user_template' => 'Write about {{topic}}',
            'input_data' => ['topic' => 'PHP testing'],
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(Mockery::any(), 'Write about PHP testing', Mockery::any())
            ->once()
            ->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('leaves unmatched placeholders unchanged', function () {
        $task = mockContentTask([
            'workspace' => null,
            'user_template' => 'Hello {{name}}, your role is {{role}}',
            'input_data' => ['name' => 'Bob'], // {{role}} has no value
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(Mockery::any(), 'Hello Bob, your role is {{role}}', Mockery::any())
            ->once()
            ->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('serialises array values as JSON in placeholders', function () {
        $task = mockContentTask([
            'workspace' => null,
            'user_template' => 'Data: {{items}}',
            'input_data' => ['items' => ['a', 'b', 'c']],
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(Mockery::any(), 'Data: ["a","b","c"]', Mockery::any())
            ->once()
            ->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });

    it('handles empty input_data without error', function () {
        $task = mockContentTask([
            'workspace' => null,
            'user_template' => 'Static template with no variables',
            'input_data' => [],
        ]);
        $task->shouldReceive('markProcessing')->once();
        $task->shouldReceive('markCompleted')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->with(Mockery::any(), 'Static template with no variables', Mockery::any())
            ->once()
            ->andReturn(mockAgenticResponse());

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $job = new ProcessContentTask($task);
        $job->handle($ai, $entitlements);
    });
});

// =========================================================================
// Retry Logic Tests
// =========================================================================

describe('retry logic', function () {
    it('job can be re-dispatched after failure', function () {
        Queue::fake();

        $task = mockContentTask();

        ProcessContentTask::dispatch($task);
        ProcessContentTask::dispatch($task); // simulated retry

        Queue::assertPushed(ProcessContentTask::class, 2);
    });

    it('failed() is called when an unhandled exception propagates', function () {
        $task = mockContentTask(['workspace' => null]);
        $task->shouldReceive('markProcessing')->once();

        $provider = Mockery::mock(AgenticProviderInterface::class);
        $provider->shouldReceive('isAvailable')->andReturn(true);
        $provider->shouldReceive('generate')
            ->andThrow(new RuntimeException('API timeout'));

        $ai = Mockery::mock(AgenticManager::class);
        $ai->shouldReceive('provider')->andReturn($provider);

        $entitlements = Mockery::mock('Core\Tenant\Services\EntitlementService');

        $task->shouldReceive('markFailed')
            ->once()
            ->with('API timeout');

        $job = new ProcessContentTask($task);

        try {
            $job->handle($ai, $entitlements);
        } catch (Throwable $e) {
            $job->failed($e);
        }
    });
});
