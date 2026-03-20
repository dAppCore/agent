<?php

declare(strict_types=1);

/**
 * Tests for the GeminiService AI provider.
 *
 * Uses mocked HTTP responses to test the service without real API calls.
 * Covers provider configuration, API key management, request handling, and responses.
 */

use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Mod\Agentic\Services\GeminiService;
use Illuminate\Support\Facades\Http;
use RuntimeException;

const GEMINI_API_URL = 'https://generativelanguage.googleapis.com/v1beta/models';

// =========================================================================
// Service Configuration Tests
// =========================================================================

describe('provider configuration', function () {
    it('returns gemini as the provider name', function () {
        $service = new GeminiService('test-api-key');

        expect($service->name())->toBe('gemini');
    });

    it('returns configured model as default model', function () {
        $service = new GeminiService('test-api-key', 'gemini-1.5-pro');

        expect($service->defaultModel())->toBe('gemini-1.5-pro');
    });

    it('uses flash as default model when not specified', function () {
        $service = new GeminiService('test-api-key');

        expect($service->defaultModel())->toBe('gemini-2.0-flash');
    });
});

// =========================================================================
// API Key Management Tests
// =========================================================================

describe('API key management', function () {
    it('reports available when API key is provided', function () {
        $service = new GeminiService('test-api-key');

        expect($service->isAvailable())->toBeTrue();
    });

    it('reports unavailable when API key is empty', function () {
        $service = new GeminiService('');

        expect($service->isAvailable())->toBeFalse();
    });

    it('sends API key in query parameter', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key-123');
        $service->generate('System', 'User');

        Http::assertSent(function ($request) {
            return str_contains($request->url(), 'key=test-api-key-123');
        });
    });
});

// =========================================================================
// Request Handling Tests
// =========================================================================

describe('request handling', function () {
    it('sends correct request body structure', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $service->generate('System prompt', 'User prompt');

        Http::assertSent(function ($request) {
            $body = $request->data();

            return $body['systemInstruction']['parts'][0]['text'] === 'System prompt'
                && $body['contents'][0]['parts'][0]['text'] === 'User prompt'
                && $body['generationConfig']['maxOutputTokens'] === 4096
                && $body['generationConfig']['temperature'] === 1.0;
        });
    });

    it('includes model name in URL', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key', 'gemini-1.5-pro');
        $service->generate('System', 'User');

        Http::assertSent(function ($request) {
            return str_contains($request->url(), 'gemini-1.5-pro:generateContent');
        });
    });

    it('applies custom configuration overrides', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $service->generate('System', 'User', [
            'model' => 'gemini-1.5-pro',
            'max_tokens' => 8192,
            'temperature' => 0.5,
        ]);

        Http::assertSent(function ($request) {
            $body = $request->data();

            return str_contains($request->url(), 'gemini-1.5-pro')
                && $body['generationConfig']['maxOutputTokens'] === 8192
                && $body['generationConfig']['temperature'] === 0.5;
        });
    });

    it('uses streamGenerateContent endpoint for streaming', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response('', 200),
        ]);

        $service = new GeminiService('test-api-key');
        iterator_to_array($service->stream('System', 'User'));

        Http::assertSent(function ($request) {
            return str_contains($request->url(), ':streamGenerateContent');
        });
    });
});

// =========================================================================
// Response Handling Tests
// =========================================================================

describe('response handling', function () {
    it('returns AgenticResponse with parsed content', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    [
                        'content' => [
                            'parts' => [['text' => 'Hello, world!']],
                        ],
                        'finishReason' => 'STOP',
                    ],
                ],
                'usageMetadata' => [
                    'promptTokenCount' => 10,
                    'candidatesTokenCount' => 5,
                ],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('You are helpful.', 'Say hello');

        expect($response)
            ->toBeInstanceOf(AgenticResponse::class)
            ->and($response->content)->toBe('Hello, world!')
            ->and($response->model)->toBe('gemini-2.0-flash')
            ->and($response->inputTokens)->toBe(10)
            ->and($response->outputTokens)->toBe(5)
            ->and($response->stopReason)->toBe('STOP');
    });

    it('tracks request duration in milliseconds', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->durationMs)
            ->toBeInt()
            ->toBeGreaterThanOrEqual(0);
    });

    it('includes raw API response for debugging', function () {
        $rawResponse = [
            'candidates' => [
                ['content' => ['parts' => [['text' => 'Response']]]],
            ],
            'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
        ];

        Http::fake([
            GEMINI_API_URL.'/*' => Http::response($rawResponse, 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->raw)
            ->toHaveKey('candidates')
            ->toHaveKey('usageMetadata');
    });

    it('returns generator for streaming responses', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response('', 200),
        ]);

        $service = new GeminiService('test-api-key');
        $generator = $service->stream('System', 'User');

        expect($generator)->toBeInstanceOf(Generator::class);
    });
});

// =========================================================================
// Edge Case Tests
// =========================================================================

describe('edge cases', function () {
    it('handles empty candidates array gracefully', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 0],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('');
    });

    it('handles missing usage metadata gracefully', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->inputTokens)->toBe(0)
            ->and($response->outputTokens)->toBe(0);
    });

    it('handles missing finish reason gracefully', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => [['text' => 'Response']]]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->stopReason)->toBeNull();
    });

    it('handles empty parts array gracefully', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'candidates' => [
                    ['content' => ['parts' => []]],
                ],
                'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 0],
            ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('');
    });
});

// =========================================================================
// Error Handling and Retry Tests
// =========================================================================

describe('error handling', function () {
    it('throws exception on client authentication error', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response([
                'error' => ['message' => 'Invalid API key'],
            ], 401),
        ]);

        $service = new GeminiService('invalid-key');

        expect(fn () => $service->generate('System', 'User'))
            ->toThrow(RuntimeException::class, 'Gemini API error');
    });

    it('retries automatically on rate limit (429)', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::sequence()
                ->push(['error' => ['message' => 'Rate limited']], 429)
                ->push([
                    'candidates' => [
                        ['content' => ['parts' => [['text' => 'Success after retry']]]],
                    ],
                    'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
                ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('Success after retry');
    });

    it('retries automatically on server error (500)', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::sequence()
                ->push(['error' => ['message' => 'Server error']], 500)
                ->push([
                    'candidates' => [
                        ['content' => ['parts' => [['text' => 'Success after retry']]]],
                    ],
                    'usageMetadata' => ['promptTokenCount' => 10, 'candidatesTokenCount' => 5],
                ], 200),
        ]);

        $service = new GeminiService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('Success after retry');
    });

    it('throws exception after exhausting max retries', function () {
        Http::fake([
            GEMINI_API_URL.'/*' => Http::response(['error' => ['message' => 'Server error']], 500),
        ]);

        $service = new GeminiService('test-api-key');

        expect(fn () => $service->generate('System', 'User'))
            ->toThrow(RuntimeException::class);
    });
});
