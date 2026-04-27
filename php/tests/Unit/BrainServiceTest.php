<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\Http;

function qdrantHeaderBrainService(): BrainService
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

afterEach(function (): void {
    Config::set('mcp.brain.qdrant.api_key', null);
});

test('BrainService_qdrantUpsert_Good_sends_api_key_header_when_configured', function (): void {
    Config::set('mcp.brain.qdrant.api_key', 'qdrant-secret');

    Http::fake([
        'https://qdrant.test/collections/openbrain/points' => Http::response(['status' => 'ok']),
    ]);

    qdrantHeaderBrainService()->qdrantUpsert([
        ['id' => 'memory-1', 'vector' => [0.1, 0.2], 'payload' => ['type' => 'note']],
    ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points'
        && $request->method() === 'PUT'
        && $request->hasHeader('api-key', 'qdrant-secret'));
});

test('BrainService_qdrantUpsert_Bad_omits_api_key_header_when_unset', function (): void {
    Config::set('mcp.brain.qdrant.api_key', null);

    Http::fake([
        'https://qdrant.test/collections/openbrain/points' => Http::response(['status' => 'ok']),
    ]);

    qdrantHeaderBrainService()->qdrantUpsert([
        ['id' => 'memory-2', 'vector' => [0.3, 0.4], 'payload' => ['type' => 'note']],
    ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points'
        && $request->method() === 'PUT'
        && ! $request->hasHeader('api-key'));
});

test('BrainService_qdrantUpsert_Ugly_treats_empty_api_key_as_unset', function (): void {
    Config::set('mcp.brain.qdrant.api_key', '');

    Http::fake([
        'https://qdrant.test/collections/openbrain/points' => Http::response(['status' => 'ok']),
    ]);

    qdrantHeaderBrainService()->qdrantUpsert([
        ['id' => 'memory-3', 'vector' => [0.5, 0.6], 'payload' => ['type' => 'note']],
    ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points'
        && $request->method() === 'PUT'
        && ! $request->hasHeader('api-key'));
});
