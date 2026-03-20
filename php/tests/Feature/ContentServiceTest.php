<?php

use Core\Mod\Agentic\Services\AgenticManager;
use Core\Mod\Agentic\Services\AgenticProviderInterface;
use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Mod\Agentic\Services\ContentService;
use Illuminate\Support\Facades\File;

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
    $this->manager = Mockery::mock(AgenticManager::class);
    $this->service = new ContentService($this->manager);
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
    $provider->shouldReceive('generate')->andThrow(new \Exception('API Error'));

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    // Create a temporary test batch file
    $testBatchPath = base_path('app/Mod/Agentic/Resources/tasks/batch-test-error.md');
    // Ensure the prompts directory exists for the test if it's looking for a template
    $promptPath = base_path('app/Mod/Agentic/Resources/prompts/content/help-article.md');

    // We need to ensure the help-article prompt exists, otherwise it fails before hitting the API
    if (! File::exists($promptPath)) {
        $this->markTestSkipped('Help article prompt not found');
    }

    File::put($testBatchPath, "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: test-slug-error\nTITLE: Test\n```");

    // Clean up potential leftover draft and state files
    $draftPath = base_path('app/Mod/Agentic/Resources/drafts/help/general/test-slug-error.md');
    $statePath = base_path('app/Mod/Agentic/Resources/tasks/batch-test-error.progress.json');
    if (File::exists($draftPath)) {
        File::delete($draftPath);
    }
    if (File::exists($statePath)) {
        File::delete($statePath);
    }

    try {
        $results = $this->service->generateBatch('batch-test-error', 'gemini', false);

        expect($results['failed'])->toBe(1);
        expect($results['articles']['test-slug-error']['status'])->toBe('failed');
        expect($results['articles']['test-slug-error']['error'])->toBe('API Error');
    } finally {
        if (File::exists($testBatchPath)) {
            File::delete($testBatchPath);
        }
        if (File::exists($draftPath)) {
            File::delete($draftPath);
        }
        if (File::exists($statePath)) {
            File::delete($statePath);
        }
    }
});

it('returns null progress when no state file exists', function () {
    $progress = $this->service->loadBatchProgress('batch-nonexistent-xyz');

    expect($progress)->toBeNull();
});

it('saves progress state after batch generation', function () {
    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')->andThrow(new \Exception('API Error'));

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $promptPath = base_path('app/Mod/Agentic/Resources/prompts/content/help-article.md');
    if (! File::exists($promptPath)) {
        $this->markTestSkipped('Help article prompt not found');
    }

    $batchId = 'batch-test-progress';
    $batchPath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.md");
    $statePath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.progress.json");

    File::put($batchPath, "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: progress-slug-a\nTITLE: Test A\n```\n### Article 2:\n```yaml\nSLUG: progress-slug-b\nTITLE: Test B\n```");

    try {
        $this->service->generateBatch($batchId, 'gemini', false, 0);

        $progress = $this->service->loadBatchProgress($batchId);

        expect($progress)->toBeArray();
        expect($progress['batch_id'])->toBe($batchId);
        expect($progress['provider'])->toBe('gemini');
        expect($progress['articles'])->toHaveKeys(['progress-slug-a', 'progress-slug-b']);
        expect($progress['articles']['progress-slug-a']['status'])->toBe('failed');
        expect($progress['articles']['progress-slug-a']['attempts'])->toBe(1);
        expect($progress['articles']['progress-slug-a']['last_error'])->toBe('API Error');
    } finally {
        File::deleteDirectory(base_path('app/Mod/Agentic/Resources/drafts/help/general'), true);
        if (File::exists($batchPath)) {
            File::delete($batchPath);
        }
        if (File::exists($statePath)) {
            File::delete($statePath);
        }
    }
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

    $promptPath = base_path('app/Mod/Agentic/Resources/prompts/content/help-article.md');
    if (! File::exists($promptPath)) {
        $this->markTestSkipped('Help article prompt not found');
    }

    $batchId = 'batch-test-resume-skip';
    $batchPath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.md");
    $statePath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.progress.json");
    $draftDir = base_path('app/Mod/Agentic/Resources/drafts/help/general');

    File::put($batchPath, "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: resume-skip-slug-a\nTITLE: Test A\n```\n### Article 2:\n```yaml\nSLUG: resume-skip-slug-b\nTITLE: Test B\n```");

    try {
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
    } finally {
        foreach (['resume-skip-slug-a', 'resume-skip-slug-b'] as $slug) {
            $draft = "{$draftDir}/{$slug}.md";
            if (File::exists($draft)) {
                File::delete($draft);
            }
        }
        if (File::exists($batchPath)) {
            File::delete($batchPath);
        }
        if (File::exists($statePath)) {
            File::delete($statePath);
        }
    }
});

it('resume returns error when no prior state exists', function () {
    $result = $this->service->resumeBatch('batch-no-state-xyz');

    expect($result)->toHaveKey('error');
    expect($result['error'])->toContain('No progress state found');
});

it('resume retries only failed and pending articles', function () {
    $slugs = ['resume-retry-a', 'resume-retry-b'];
    $callCount = 0;

    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')
        ->andReturnUsing(function () use (&$callCount) {
            $callCount++;

            // Call 1: A on first run → fails
            // Call 2: B on first run → succeeds
            // Resume run: only A is retried (B is already generated)
            if ($callCount === 1) {
                throw new \Exception('Transient Error');
            }

            return makeAgenticResponse('## Content');
        });

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $promptPath = base_path('app/Mod/Agentic/Resources/prompts/content/help-article.md');
    if (! File::exists($promptPath)) {
        $this->markTestSkipped('Help article prompt not found');
    }

    $batchId = 'batch-test-resume-retry';
    $batchPath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.md");
    $statePath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.progress.json");
    $draftDir = base_path('app/Mod/Agentic/Resources/drafts/help/general');

    File::put($batchPath, "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: resume-retry-a\nTITLE: Retry A\n```\n### Article 2:\n```yaml\nSLUG: resume-retry-b\nTITLE: Retry B\n```");

    try {
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
    } finally {
        foreach ($slugs as $slug) {
            $draft = "{$draftDir}/{$slug}.md";
            if (File::exists($draft)) {
                File::delete($draft);
            }
        }
        if (File::exists($batchPath)) {
            File::delete($batchPath);
        }
        if (File::exists($statePath)) {
            File::delete($statePath);
        }
    }
});

it('retries individual failures up to maxRetries times', function () {
    $callCount = 0;
    $provider = Mockery::mock(AgenticProviderInterface::class);
    $provider->shouldReceive('generate')
        ->andReturnUsing(function () use (&$callCount) {
            $callCount++;
            if ($callCount < 3) {
                throw new \Exception("Attempt {$callCount} failed");
            }

            return makeAgenticResponse('## Content');
        });

    $this->manager->shouldReceive('provider')->with('gemini')->andReturn($provider);

    $promptPath = base_path('app/Mod/Agentic/Resources/prompts/content/help-article.md');
    if (! File::exists($promptPath)) {
        $this->markTestSkipped('Help article prompt not found');
    }

    $batchId = 'batch-test-maxretries';
    $batchPath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.md");
    $statePath = base_path("app/Mod/Agentic/Resources/tasks/{$batchId}.progress.json");
    $draftPath = base_path('app/Mod/Agentic/Resources/drafts/help/general/maxretries-slug.md');

    File::put($batchPath, "# Test Batch\n**Service:** Test\n### Article 1:\n```yaml\nSLUG: maxretries-slug\nTITLE: Retry Test\n```");

    try {
        // With maxRetries=2 (3 total attempts), succeeds on 3rd attempt
        $results = $this->service->generateBatch($batchId, 'gemini', false, 2);

        expect($results['generated'])->toBe(1);
        expect($results['failed'])->toBe(0);
        expect($results['articles']['maxretries-slug']['status'])->toBe('generated');
        expect($callCount)->toBe(3);

        $progress = $this->service->loadBatchProgress($batchId);
        expect($progress['articles']['maxretries-slug']['status'])->toBe('generated');
        expect($progress['articles']['maxretries-slug']['attempts'])->toBe(3);
    } finally {
        if (File::exists($batchPath)) {
            File::delete($batchPath);
        }
        if (File::exists($statePath)) {
            File::delete($statePath);
        }
        if (File::exists($draftPath)) {
            File::delete($draftPath);
        }
    }
});
