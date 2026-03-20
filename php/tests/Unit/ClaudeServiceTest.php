<?php

declare(strict_types=1);

/**
 * Tests for the ClaudeService AI provider.
 *
 * Uses mocked HTTP responses to test the service without real API calls.
 * Covers provider configuration, API key management, request handling, and responses.
 */

use Core\Mod\Agentic\Services\AgenticResponse;
use Core\Mod\Agentic\Services\ClaudeService;
use Illuminate\Http\Client\ConnectionException;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use RuntimeException;

const CLAUDE_API_URL = 'https://api.anthropic.com/v1/messages';

// =========================================================================
// Service Configuration Tests
// =========================================================================

describe('provider configuration', function () {
    it('returns claude as the provider name', function () {
        $service = new ClaudeService('test-api-key');

        expect($service->name())->toBe('claude');
    });

    it('returns configured model as default model', function () {
        $service = new ClaudeService('test-api-key', 'claude-opus-4-20250514');

        expect($service->defaultModel())->toBe('claude-opus-4-20250514');
    });

    it('uses sonnet as default model when not specified', function () {
        $service = new ClaudeService('test-api-key');

        expect($service->defaultModel())->toBe('claude-sonnet-4-20250514');
    });
});

// =========================================================================
// API Key Management Tests
// =========================================================================

describe('API key management', function () {
    it('reports available when API key is provided', function () {
        $service = new ClaudeService('test-api-key');

        expect($service->isAvailable())->toBeTrue();
    });

    it('reports unavailable when API key is empty', function () {
        $service = new ClaudeService('');

        expect($service->isAvailable())->toBeFalse();
    });

    it('sends API key in x-api-key header', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-sonnet-4-20250514',
                'content' => [['type' => 'text', 'text' => 'Response']],
                'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key-123');
        $service->generate('System', 'User');

        Http::assertSent(function ($request) {
            return $request->hasHeader('x-api-key', 'test-api-key-123')
                && $request->hasHeader('anthropic-version', '2023-06-01')
                && $request->hasHeader('content-type', 'application/json');
        });
    });
});

// =========================================================================
// Request Handling Tests
// =========================================================================

describe('request handling', function () {
    it('sends correct request body structure', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-sonnet-4-20250514',
                'content' => [['type' => 'text', 'text' => 'Response']],
                'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $service->generate('System prompt', 'User prompt');

        Http::assertSent(function ($request) {
            $body = $request->data();

            return $body['system'] === 'System prompt'
                && $body['messages'][0]['role'] === 'user'
                && $body['messages'][0]['content'] === 'User prompt'
                && $body['model'] === 'claude-sonnet-4-20250514'
                && $body['max_tokens'] === 4096
                && $body['temperature'] === 1.0;
        });
    });

    it('applies custom configuration overrides', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-opus-4-20250514',
                'content' => [['type' => 'text', 'text' => 'Response']],
                'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $service->generate('System', 'User', [
            'model' => 'claude-opus-4-20250514',
            'max_tokens' => 8192,
            'temperature' => 0.5,
        ]);

        Http::assertSent(function ($request) {
            $body = $request->data();

            return $body['model'] === 'claude-opus-4-20250514'
                && $body['max_tokens'] === 8192
                && $body['temperature'] === 0.5;
        });
    });

    it('sends stream flag for streaming requests', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response('', 200),
        ]);

        $service = new ClaudeService('test-api-key');
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
            CLAUDE_API_URL => Http::response([
                'id' => 'msg_123',
                'type' => 'message',
                'role' => 'assistant',
                'model' => 'claude-sonnet-4-20250514',
                'content' => [
                    ['type' => 'text', 'text' => 'Hello, world!'],
                ],
                'stop_reason' => 'end_turn',
                'usage' => [
                    'input_tokens' => 10,
                    'output_tokens' => 5,
                ],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('You are helpful.', 'Say hello');

        expect($response)
            ->toBeInstanceOf(AgenticResponse::class)
            ->and($response->content)->toBe('Hello, world!')
            ->and($response->model)->toBe('claude-sonnet-4-20250514')
            ->and($response->inputTokens)->toBe(10)
            ->and($response->outputTokens)->toBe(5)
            ->and($response->stopReason)->toBe('end_turn');
    });

    it('tracks request duration in milliseconds', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-sonnet-4-20250514',
                'content' => [['type' => 'text', 'text' => 'Response']],
                'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->durationMs)
            ->toBeInt()
            ->toBeGreaterThanOrEqual(0);
    });

    it('includes raw API response for debugging', function () {
        $rawResponse = [
            'id' => 'msg_123',
            'model' => 'claude-sonnet-4-20250514',
            'content' => [['type' => 'text', 'text' => 'Response']],
            'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
        ];

        Http::fake([
            CLAUDE_API_URL => Http::response($rawResponse, 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->raw['id'])->toBe('msg_123');
    });

    it('returns generator for streaming responses', function () {
        $stream = "data: {\"type\": \"content_block_delta\", \"delta\": {\"text\": \"Hello\"}}\n\n";
        $stream .= "data: {\"type\": \"content_block_delta\", \"delta\": {\"text\": \" world\"}}\n\n";
        $stream .= "data: [DONE]\n\n";

        Http::fake([
            CLAUDE_API_URL => Http::response($stream, 200, ['Content-Type' => 'text/event-stream']),
        ]);

        $service = new ClaudeService('test-api-key');
        $generator = $service->stream('System', 'User');

        expect($generator)->toBeInstanceOf(Generator::class);
    });
});

// =========================================================================
// Edge Case Tests
// =========================================================================

describe('edge cases', function () {
    it('handles empty content array gracefully', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-sonnet-4-20250514',
                'content' => [],
                'usage' => ['input_tokens' => 10, 'output_tokens' => 0],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('');
    });

    it('handles missing usage data gracefully', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-sonnet-4-20250514',
                'content' => [['type' => 'text', 'text' => 'Response']],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->inputTokens)->toBe(0)
            ->and($response->outputTokens)->toBe(0);
    });

    it('handles missing stop reason gracefully', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'model' => 'claude-sonnet-4-20250514',
                'content' => [['type' => 'text', 'text' => 'Response']],
                'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
            ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->stopReason)->toBeNull();
    });
});

// =========================================================================
// Error Handling and Retry Tests
// =========================================================================

describe('error handling', function () {
    it('throws exception on client authentication error', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response([
                'error' => ['message' => 'Invalid API key'],
            ], 401),
        ]);

        $service = new ClaudeService('invalid-key');

        expect(fn () => $service->generate('System', 'User'))
            ->toThrow(RuntimeException::class, 'Claude API error');
    });

    it('retries automatically on rate limit (429)', function () {
        Http::fake([
            CLAUDE_API_URL => Http::sequence()
                ->push(['error' => ['message' => 'Rate limited']], 429)
                ->push([
                    'model' => 'claude-sonnet-4-20250514',
                    'content' => [['type' => 'text', 'text' => 'Success after retry']],
                    'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
                ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('Success after retry');
    });

    it('retries automatically on server error (500)', function () {
        Http::fake([
            CLAUDE_API_URL => Http::sequence()
                ->push(['error' => ['message' => 'Server error']], 500)
                ->push([
                    'model' => 'claude-sonnet-4-20250514',
                    'content' => [['type' => 'text', 'text' => 'Success after retry']],
                    'usage' => ['input_tokens' => 10, 'output_tokens' => 5],
                ], 200),
        ]);

        $service = new ClaudeService('test-api-key');
        $response = $service->generate('System', 'User');

        expect($response->content)->toBe('Success after retry');
    });

    it('throws exception after exhausting max retries', function () {
        Http::fake([
            CLAUDE_API_URL => Http::response(['error' => ['message' => 'Server error']], 500),
        ]);

        $service = new ClaudeService('test-api-key');

        expect(fn () => $service->generate('System', 'User'))
            ->toThrow(RuntimeException::class);
    });
});

// =========================================================================
// Stream Error Handling Tests
// =========================================================================

describe('stream error handling', function () {
    it('yields error event when connection fails', function () {
        Http::fake(function () {
            throw new ConnectionException('Connection refused');
        });

        $service = new ClaudeService('test-api-key');
        $results = iterator_to_array($service->stream('System', 'User'));

        expect($results)->toHaveCount(1)
            ->and($results[0])->toBeArray()
            ->and($results[0]['type'])->toBe('error')
            ->and($results[0]['message'])->toContain('Connection refused');
    });

    it('yields error event when request throws a runtime exception', function () {
        Http::fake(function () {
            throw new RuntimeException('Unexpected failure');
        });

        $service = new ClaudeService('test-api-key');
        $results = iterator_to_array($service->stream('System', 'User'));

        expect($results)->toHaveCount(1)
            ->and($results[0]['type'])->toBe('error')
            ->and($results[0]['message'])->toBe('Unexpected failure');
    });

    it('error event contains type and message keys', function () {
        Http::fake(function () {
            throw new RuntimeException('Stream broke');
        });

        $service = new ClaudeService('test-api-key');
        $event = iterator_to_array($service->stream('System', 'User'))[0];

        expect($event)->toHaveKeys(['type', 'message'])
            ->and($event['type'])->toBe('error');
    });

    it('logs stream errors', function () {
        Log::spy();

        Http::fake(function () {
            throw new RuntimeException('Logging test error');
        });

        $service = new ClaudeService('test-api-key');
        iterator_to_array($service->stream('System', 'User'));

        Log::shouldHaveReceived('error')
            ->with('Claude stream error', \Mockery::on(fn ($ctx) => str_contains($ctx['message'], 'Logging test error')))
            ->once();
    });

    it('yields text chunks normally when no error occurs', function () {
        $stream = "data: {\"type\": \"content_block_delta\", \"delta\": {\"text\": \"Hello\"}}\n\n";
        $stream .= "data: {\"type\": \"content_block_delta\", \"delta\": {\"text\": \" world\"}}\n\n";
        $stream .= "data: [DONE]\n\n";

        Http::fake([
            CLAUDE_API_URL => Http::response($stream, 200, ['Content-Type' => 'text/event-stream']),
        ]);

        $service = new ClaudeService('test-api-key');
        $results = iterator_to_array($service->stream('System', 'User'));

        expect($results)->toBe(['Hello', ' world']);
    });
});
