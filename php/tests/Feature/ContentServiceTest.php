<?php

use Core\Mod\Agentic\Services\AgenticManager;
use Core\Mod\Agentic\Services\AgenticProviderInterface;
use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Mod\Agentic\Services\ContentService;
use Illuminate\Support\Facades\File;

/*
|--------------------------------------------------------------------------
| Content fixtures
|--------------------------------------------------------------------------
|
| ContentService resolves every path it touches through base_path() plus a
| configurable relative prefix, defaulting to app/Mod/Agentic/Resources/*.
| That default belongs to a host application. Under Testbench base_path() is
| the bare skeleton, so the help-article prompt template was never there and
| five of these tests did nothing but markTestSkipped('Help article prompt
| not found') — permanently, on every machine and in CI. The batch fixture
| batch-001-link-getting-started was missing for the same reason, which is
| why the three tests that read it failed rather than skipped.
|
| The three paths are config-driven, so the suite points them at a sandbox
| under base_path() and writes the fixtures it needs. Nothing here depends on
| a host application any more.
|
*/

const CONTENT_SANDBOX = 'content-service-sandbox';

function contentSandbox(string $relative = ''): string
{
    return base_path(rtrim(CONTENT_SANDBOX.'/'.ltrim($relative, '/'), '/'));
}

function contentTask(string $file): string
{
    return contentSandbox('tasks/'.$file);
}

function makeAgenticResponse(string $content = '## Article Content'): AgenticResponse
{
    return new AgenticResponse(
        content: $content,
        model: 'test-model',
        inputTokens: 0,
        outputTokens: 0,
        durationMs: 0,
    );
}

beforeEach(function () {
    config([
        'mcp.content.batch_path' => CONTENT_SANDBOX.'/tasks',
        'mcp.content.prompt_path' => CONTENT_SANDBOX.'/prompts/content',
        'mcp.content.drafts_path' => CONTENT_SANDBOX.'/drafts',
    ]);

    File::deleteDirectory(contentSandbox());
    File::ensureDirectoryExists(contentSandbox('tasks'));
    File::ensureDirectoryExists(contentSandbox('prompts/content'));

    File::put(
        contentSandbox('prompts/content/help-article.md'),
        "# Help Article\n\nWrite an article titled {{TITLE}} for {{SERVICE_NAME}}.\n",
    );

    File::put(contentTask('batch-001-link-getting-started.md'), <<<'MARKDOWN'
        # Batch 001 — Host Link getting started

        **Service:** Host Link
        **Category:** Getting Started
        **Priority:** high

        ### Article 1:
        ```yaml
        SLUG: link-getting-started
        TITLE: Getting started with Host Link
        ```

        ### Article 2:
        ```yaml
        SLUG: link-connect-a-domain
        TITLE: Connect a domain to Host Link
        ```
        MARKDOWN);

    // config() is read in the constructor, so the service is built after it.
    $this->manager = Mockery::mock(AgenticManager::class);
    $this->service = new ContentService($this->manager);
});

afterEach(function () {
    File::deleteDirectory(contentSandbox());
});

it('lists available batches', function () {
    $batches = $this->service->listBatches();

    expect($batches)->toBeArray();
    expect(count($batches))->toBeGreaterThan(0);
    // Check the first batch found
    $firstBatch = collect($batches)->firstWhere('id', 'batch-001-link-getting-started');
    expect($firstBatch)->not->toBeNull();
    expect($firstBatch)->toHaveKeys(['id', 'service', 'category', 'article_count']);
    expect($firstBatch['service'])->toBe('Host Link');
});

it('loads a specific batch', function () {
    $batch = $this->service->loadBatch('batch-001-link-getting-started');

    expect($batch)->toBeArray();
    expect($batch['service'])->toBe('Host Link');
    expect($batch['articles'])->toBeArray();
    expect(count($batch['articles']))->toBeGreaterThan(0);
});

it('generates content for a batch (dry run)', function () {
    $results = $this->service->generateBatch('batch-001-link-getting-started', 'gemini', true);

    expect($results['batch_id'])->toBe('batch-001-link-getting-started');
    expect($results['articles'])->not->toBeEmpty();

    foreach ($results['articles'] as $slug => $status) {
        expect($status['status'])->toBe('would_generate');
    }
});

it('handles generation errors gracefully', function () {
    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')->andThrow(new Exception('API Error'));

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    File::put(contentTask('batch-test-error.md'), "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: test-slug-error\nTITLE: Test\n```");

    $results = $this->service->generateBatch('batch-test-error', 'gemini', false);

    expect($results['failed'])->toBe(1);
    expect($results['articles']['test-slug-error']['status'])->toBe('failed');
    expect($results['articles']['test-slug-error']['error'])->toBe('API Error');
});

it('returns null progress when no state file exists', function () {
    $progress = $this->service->loadBatchProgress('batch-nonexistent-xyz');

    expect($progress)->toBeNull();
});

it('saves progress state after batch generation', function () {
    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')->andThrow(new Exception('API Error'));

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $batchId = 'batch-test-progress';

    File::put(contentTask("{$batchId}.md"), "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: progress-slug-a\nTITLE: Test A\n```\n### Article 2:\n```yaml\nSLUG: progress-slug-b\nTITLE: Test B\n```");

    $this->service->generateBatch($batchId, 'gemini', false, 0);

    $progress = $this->service->loadBatchProgress($batchId);

    expect($progress)->toBeArray();
    expect($progress['batch_id'])->toBe($batchId);
    expect($progress['provider'])->toBe('gemini');
    expect($progress['articles'])->toHaveKeys(['progress-slug-a', 'progress-slug-b']);
    expect($progress['articles']['progress-slug-a']['status'])->toBe('failed');
    expect($progress['articles']['progress-slug-a']['attempts'])->toBe(1);
    expect($progress['articles']['progress-slug-a']['last_error'])->toBe('API Error');
});

it('skips previously generated articles on second run', function () {
    $callCount = 0;
    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')
        ->andReturnUsing(function () use (&$callCount) {
            $callCount++;

            return makeAgenticResponse();
        });

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $batchId = 'batch-test-resume-skip';

    File::put(contentTask("{$batchId}.md"), "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: resume-skip-slug-a\nTITLE: Test A\n```\n### Article 2:\n```yaml\nSLUG: resume-skip-slug-b\nTITLE: Test B\n```");

    // First run generates both articles
    $first = $this->service->generateBatch($batchId, 'gemini', false, 0);
    expect($first['generated'])->toBe(2);
    expect($callCount)->toBe(2);

    // Second run skips already-generated articles
    $second = $this->service->generateBatch($batchId, 'gemini', false, 0);
    expect($second['generated'])->toBe(0);
    expect($second['skipped'])->toBe(2);
    // Provider should not have been called again
    expect($callCount)->toBe(2);
});

it('resume returns error when no prior state exists', function () {
    $result = $this->service->resumeBatch('batch-no-state-xyz');

    expect($result)->toHaveKey('error');
    expect($result['error'])->toContain('No progress state found');
});

it('resume retries only failed and pending articles', function () {
    $callCount = 0;

    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')
        ->andReturnUsing(function () use (&$callCount) {
            $callCount++;

            // Call 1: A on first run → fails
            // Call 2: B on first run → succeeds
            // Resume run: only A is retried (B is already generated)
            if ($callCount === 1) {
                throw new Exception('Transient Error');
            }

            return makeAgenticResponse('## Content');
        });

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $batchId = 'batch-test-resume-retry';

    File::put(contentTask("{$batchId}.md"), "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: resume-retry-a\nTITLE: Retry A\n```\n### Article 2:\n```yaml\nSLUG: resume-retry-b\nTITLE: Retry B\n```");

    // First run: A fails, B succeeds
    $first = $this->service->generateBatch($batchId, 'gemini', false, 0);
    expect($first['failed'])->toBe(1);
    expect($first['generated'])->toBe(1);
    expect($first['articles']['resume-retry-a']['status'])->toBe('failed');
    expect($first['articles']['resume-retry-b']['status'])->toBe('generated');

    // Resume: only retries failed article A
    $resumed = $this->service->resumeBatch($batchId, 'gemini', 0);
    expect($resumed)->toHaveKey('resumed_from');
    expect($resumed['skipped'])->toBeGreaterThanOrEqual(1); // B is skipped
    expect($resumed['articles']['resume-retry-b']['status'])->toBe('skipped');
});

it('retries individual failures up to maxRetries times', function () {
    $callCount = 0;
    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')
        ->andReturnUsing(function () use (&$callCount) {
            $callCount++;
            if ($callCount < 3) {
                throw new Exception("Attempt {$callCount} failed");
            }

            return makeAgenticResponse('## Content');
        });

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $batchId = 'batch-test-maxretries';

    File::put(contentTask("{$batchId}.md"), "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: maxretries-slug\nTITLE: Retry Test\n```");

    // With maxRetries=2 (3 total attempts), succeeds on 3rd attempt
    $results = $this->service->generateBatch($batchId, 'gemini', false, 2);

    expect($results['generated'])->toBe(1);
    expect($results['failed'])->toBe(0);
    expect($results['articles']['maxretries-slug']['status'])->toBe('generated');
    expect($callCount)->toBe(3);

    $progress = $this->service->loadBatchProgress($batchId);
    expect($progress['articles']['maxretries-slug']['status'])->toBe('generated');
    expect($progress['articles']['maxretries-slug']['attempts'])->toBe(3);
});
