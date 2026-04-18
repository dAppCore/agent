<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Services\Concerns\HasRetry;
use Core\Mod\Agentic\Services\Concerns\HasStreamParsing;
use Generator;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use Throwable;

class ClaudeService implements AgenticProviderInterface
{
    use HasRetry;
    use HasStreamParsing;

    private const API_URL = 'https://api.anthropic.com/v1/messages';

    private const API_VERSION = '2023-06-01';

    public function __construct(
        protected string $apiKey,
        protected string $model = 'claude-sonnet-4-20250514',
    ) {}

    public function generate(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): AgenticResponse {
        $startTime = microtime(true);

        $response = $this->withRetry(
            fn () => $this->client()->post(self::API_URL, [
                'model' => $config['model'] ?? $this->model,
                'max_tokens' => $config['max_tokens'] ?? 4096,
                'temperature' => $config['temperature'] ?? 1.0,
                'system' => $systemPrompt,
                'messages' => [
                    ['role' => 'user', 'content' => $userPrompt],
                ],
            ]),
            'Claude'
        );

        $data = $response->json();
        $durationMs = (int) ((microtime(true) - $startTime) * 1000);

        return new AgenticResponse(
            content: $data['content'][0]['text'] ?? '',
            model: $data['model'],
            inputTokens: $data['usage']['input_tokens'] ?? 0,
            outputTokens: $data['usage']['output_tokens'] ?? 0,
            durationMs: $durationMs,
            stopReason: $data['stop_reason'] ?? null,
            raw: $data,
        );
    }

    /**
     * Stream a completion from Claude.
     *
     * Yields text chunks as strings on success.
     *
     * On failure, yields a single error event array and terminates:
     *   ['type' => 'error', 'message' => string]
     *
     * @return Generator<string|array{type: 'error', message: string}>
     */
    public function stream(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): Generator {
        try {
            $response = $this->client()
                ->withOptions(['stream' => true])
                ->post(self::API_URL, [
                    'model' => $config['model'] ?? $this->model,
                    'max_tokens' => $config['max_tokens'] ?? 4096,
                    'temperature' => $config['temperature'] ?? 1.0,
                    'stream' => true,
                    'system' => $systemPrompt,
                    'messages' => [
                        ['role' => 'user', 'content' => $userPrompt],
                    ],
                ]);

            yield from $this->parseSSEStream(
                $response->getBody(),
                fn (array $data) => $data['delta']['text'] ?? null
            );
        } catch (Throwable $e) {
            Log::error('Claude stream error', [
                'message' => $e->getMessage(),
                'exception' => $e,
            ]);

            yield ['type' => 'error', 'message' => $e->getMessage()];
        }
    }

    public function name(): string
    {
        return 'claude';
    }

    public function defaultModel(): string
    {
        return $this->model;
    }

    public function isAvailable(): bool
    {
        return ! empty($this->apiKey);
    }

    private function client(): PendingRequest
    {
        return Http::withHeaders([
            'x-api-key' => $this->apiKey,
            'anthropic-version' => self::API_VERSION,
            'content-type' => 'application/json',
        ])->timeout(300);
    }
}
