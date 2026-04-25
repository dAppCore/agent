<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Queue;

function rememberValidationBrainService(): BrainService
{
    return new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
        elasticsearchUrl: 'https://elasticsearch.test',
    );
}

function rememberValidationAttributes(array $attributes = []): array
{
    return array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Brain validation test memory.',
        'tags' => ['brain', 'validation'],
        'confidence' => 0.9,
        'org' => 'core',
        'project' => 'agent',
    ], $attributes);
}

test('BrainRememberValidation_remember_Good_accepts_valid_content_and_tags', function (): void {
    Queue::fake();

    $memory = rememberValidationBrainService()->remember(rememberValidationAttributes([
        'content' => 'hello world',
        'tags' => ['a', 'b'],
    ]));

    expect($memory->exists)->toBeTrue()
        ->and($memory->content)->toBe('hello world')
        ->and($memory->tags)->toBe(['a', 'b']);

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memory->id);
});

test('BrainRememberValidation_remember_Bad_rejects_content_longer_than_65536_bytes', function (): void {
    Queue::fake();

    expect(fn () => rememberValidationBrainService()->remember(rememberValidationAttributes([
        'content' => str_repeat('a', 65537),
    ])))->toThrow(\InvalidArgumentException::class, 'content exceeds maximum length of 65536 bytes');

    Queue::assertNothingPushed();
});

test('BrainRememberValidation_remember_Bad_rejects_more_than_100_tags', function (): void {
    Queue::fake();

    expect(fn () => rememberValidationBrainService()->remember(rememberValidationAttributes([
        'tags' => array_fill(0, 101, 'x'),
    ])))->toThrow(\InvalidArgumentException::class, 'tags array exceeds maximum size of 100');

    Queue::assertNothingPushed();
});

test('BrainRememberValidation_remember_Ugly_rejects_tags_longer_than_128_bytes', function (): void {
    Queue::fake();

    expect(fn () => rememberValidationBrainService()->remember(rememberValidationAttributes([
        'tags' => ['valid', str_repeat('x', 129)],
    ])))->toThrow(\InvalidArgumentException::class, 'tag at index 1 exceeds maximum length of 128');

    Queue::assertNothingPushed();
});

test('BrainRememberValidation_remember_Good_accepts_content_at_exactly_65536_bytes', function (): void {
    Queue::fake();

    $content = str_repeat('a', 65536);
    $memory = rememberValidationBrainService()->remember(rememberValidationAttributes([
        'content' => $content,
        'tags' => ['boundary'],
    ]));

    expect($memory->exists)->toBeTrue()
        ->and($memory->content)->toBe($content);

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memory->id);
});

test('BrainRememberValidation_recall_Bad_rejects_min_confidence_above_one', function (): void {
    expect(fn () => rememberValidationBrainService()->recall(
        'brain validation query',
        5,
        ['min_confidence' => 1.1],
        createWorkspace()->id,
    ))->toThrow(\InvalidArgumentException::class, 'min_confidence must be between 0.0 and 1.0');
});

test('BrainRememberValidation_forget_Bad_rejects_ids_longer_than_64_characters', function (): void {
    expect(fn () => rememberValidationBrainService()->forget(str_repeat('x', 65)))
        ->toThrow(\InvalidArgumentException::class, 'id exceeds maximum length of 64');
});
