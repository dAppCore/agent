<?php

declare(strict_types=1);

/**
 * Tests for the OpenAIService AI provider.
 *
 * Uses mocked HTTP responses to test the service without real API calls.
 * Covers provider configuration, API key management, request handling, and responses.
 */

use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Mod\Agentic\Services\OpenAIService;
use Illuminate\Support\Facades\Http;
use RuntimeException;

const OPENAI_API_URL = 'https://api.openai.com/v1/chat/completions';

// =========================================================================
// Service Configuration Tests
// =========================================================================

describe('provider configuration', function () {
    it('returns openai as the provider name', function () {
        $service = new OpenAIService('test-api-key');

        expect($service->name())->toBe('openai');
    });

    it('returns configured model as default model', function () {
        $service = new OpenAIService('test-api-key', 'gpt-4o');

        expect($service->defaultModel())->toBe('gpt-4o');
    });

    it('uses gpt-4o-mini as default model when not specified', function () {
        $service = new OpenAIService('test-api-key');

        expect($service->defaultModel())->toBe('gpt-4o-mini');
    });
});

// =========================================================================
// API Key Management Tests
// =========================================================================

describe('API key management', function () {
    it('reports available when API key is provided', function () {
        $service = new OpenAIService('test-api-key');

        expect($service->isAvailable())->toBeTrue();
    });

    it('reports unavailable when API key is empty', function () {
        $service = new OpenAIService('');

        expect($service->isAvailable())->toBeFalse();
    });

    it('sends API key in Authorization Bearer header', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [
                    ['message' => ['content' => 'Response'], 'finish_reason' => 'stop'],
                ],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key-123');
        $service->generate('System', 'User');

        Http::assertSent(function ($request) {
            return $request->hasHeader('Authorization', 'Bearer test-api-key-123')
                && $request->hasHeader('Content-Type', 'application/json');
        });
    });
});

// =========================================================================
// Request Handling Tests
// =========================================================================

describe('request handling', function () {
    it('sends correct request body structure', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [
                    ['message' => ['content' => 'Response'], 'finish_reason' => 'stop'],
                ],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $service->generate('System prompt', 'User prompt');

        Http::assertSent(function ($request) {
            $body = $request->data();

            return $body['messages'][0]['role'] === 'system'
                && $body['messages'][0]['content'] === 'System prompt'
                && $body['messages'][1]['role'] === 'user'
                && $body['messages'][1]['content'] === 'User prompt'
                && $body['model'] === 'gpt-4o-mini'
                && $body['max_tokens'] === 4096
                && $body['temperature'] === 1.0;
        });
    });

    it('applies custom configuration overrides', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o',
                'choices' => [
                    ['message' => ['content' => 'Response'], 'finish_reason' => 'stop'],
                ],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $service->generate('System', 'User', [
            'model' => 'gpt-4o',
            'max_tokens' => 8192,
            'temperature' => 0.5,
        ]);

        Http::assertSent(function ($request) {
            $body = $request->data();

            return $body['model'] === 'gpt-4o'
                && $body['max_tokens'] === 8192
                && $body['temperature'] === 0.5;
        });
    });

    it('sends stream flag for streaming requests', function () {
        Http::fake([
            OPENAI_API_URL => Http::response('', 200),
        ]);

        $service = new OpenAIService('test-api-key');
        iterator_to_array($service->stream('System', 'User'));

        Http::assertSent(function ($request) {
            return $request->data()['stream'] === true;
        });
    });
});

// =========================================================================
// Response Handling Tests
// =========================================================================

describe('response handling', function () {
    it('returns AgenticResponse with parsed content', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'id' => 'chatcmpl-123',
                'object' => 'chat.completion',
                'model' => 'gpt-4o-mini',
                'choices' => [
                    [
                        'index' => 0,
                        'message' => [
                            'role' => 'assistant',
                            'content' => 'Hello, world!',
                        ],
                        'finish_reason' => 'stop',
                    ],
                ],
                'usage' => [
                    'prompt_tokens' => 10,
                    'completion_tokens' => 5,
                    'total_tokens' => 15,
                ],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('You are helpful.', 'Say hello');

        expect($response)
            ->toBeInstanceOf(AgenticResponse::class)
            ->and($response->content)->toBe('Hello, world!')
            ->and($response->model)->toBe('gpt-4o-mini')
            ->and($response->inputTokens)->toBe(10)
            ->and($response->outputTokens)->toBe(5)
            ->and($response->stopReason)->toBe('stop');
    });

    it('tracks request duration in milliseconds', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [
                    ['message' => ['content' => 'Response'], 'finish_reason' => 'stop'],
                ],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->durationMs)
            ->toBeInt()
            ->toBeGreaterThanOrEqual(0);
    });

    it('includes raw API response for debugging', function () {
        $rawResponse = [
            'id' => 'chatcmpl-123',
            'model' => 'gpt-4o-mini',
            'choices' => [
                ['message' => ['content' => 'Response'], 'finish_reason' => 'stop'],
            ],
            'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
        ];

        Http::fake([
            OPENAI_API_URL => Http::response($rawResponse, 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->raw['id'])->toBe('chatcmpl-123');
    });

    it('returns generator for streaming responses', function () {
        $stream = "data: {\"choices\": [{\"delta\": {\"content\": \"Hello\"}}]}\n\n";
        $stream .= "data: {\"choices\": [{\"delta\": {\"content\": \" world\"}}]}\n\n";
        $stream .= "data: [DONE]\n\n";

        Http::fake([
            OPENAI_API_URL => Http::response($stream, 200, ['Content-Type' => 'text/event-stream']),
        ]);

        $service = new OpenAIService('test-api-key');
        $generator = $service->stream('System', 'User');

        expect($generator)->toBeInstanceOf(Generator::class);
    });
});

// =========================================================================
// Edge Case Tests
// =========================================================================

describe('edge cases', function () {
    it('handles empty choices array gracefully', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 0],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('');
    });

    it('handles missing usage data gracefully', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [
                    ['message' => ['content' => 'Response'], 'finish_reason' => 'stop'],
                ],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->inputTokens)->toBe(0)
            ->and($response->outputTokens)->toBe(0);
    });

    it('handles missing finish reason gracefully', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [
                    ['message' => ['content' => 'Response']],
                ],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->stopReason)->toBeNull();
    });

    it('handles null content gracefully', function () {
        Http::fake([
            OPENAI_API_URL => Http::response([
                'model' => 'gpt-4o-mini',
                'choices' => [
                    ['message' => ['content' => null], 'finish_reason' => 'stop'],
                ],
                'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 0],
            ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
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
            OPENAI_API_URL => Http::response([
                'error' => ['message' => 'Invalid API key'],
            ], 401),
        ]);

        $service = new OpenAIService('invalid-key');

        expect(fn () => $service->generate('System', 'User'))
            ->toThrow(RuntimeException::class, 'OpenAI API error');
    });

    it('retries automatically on rate limit (429)', function () {
        Http::fake([
            OPENAI_API_URL => Http::sequence()
                ->push(['error' => ['message' => 'Rate limited']], 429)
                ->push([
                    'model' => 'gpt-4o-mini',
                    'choices' => [
                        ['message' => ['content' => 'Success after retry'], 'finish_reason' => 'stop'],
                    ],
                    'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
                ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('Success after retry');
    });

    it('retries automatically on server error (500)', function () {
        Http::fake([
            OPENAI_API_URL => Http::sequence()
                ->push(['error' => ['message' => 'Server error']], 500)
                ->push([
                    'model' => 'gpt-4o-mini',
                    'choices' => [
                        ['message' => ['content' => 'Success after retry'], 'finish_reason' => 'stop'],
                    ],
                    'usage' => ['prompt_tokens' => 10, 'completion_tokens' => 5],
                ], 200),
        ]);

        $service = new OpenAIService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('Success after retry');
    });

    it('throws exception after exhausting max retries', function () {
        Http::fake([
            OPENAI_API_URL => Http::response(['error' => ['message' => 'Server error']], 500),
        ]);

        $service = new OpenAIService('test-api-key');

        expect(fn () => $service->generate('System', 'User'))
            ->toThrow(RuntimeException::class);
    });
});
