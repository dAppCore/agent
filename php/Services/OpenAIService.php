<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Services\Concerns\HasRetry;
use Core\Mod\Agentic\Services\Concerns\HasStreamParsing;
use Generator;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;

class OpenAIService implements AgenticProviderInterface
{
    use HasRetry;
    use HasStreamParsing;

    private const API_URL = 'https://api.openai.com/v1/chat/completions';

    public function __construct(
        protected string $apiKey,
        protected string $model = 'gpt-4o-mini',
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
                'messages' => [
                    ['role' => 'system', 'content' => $systemPrompt],
                    ['role' => 'user', 'content' => $userPrompt],
                ],
            ]),
            'OpenAI'
        );

        $data = $response->json();
        $durationMs = (int) ((microtime(true) - $startTime) * 1000);

        return new AgenticResponse(
            content: $data['choices'][0]['message']['content'] ?? '',
            model: $data['model'],
            inputTokens: $data['usage']['prompt_tokens'] ?? 0,
            outputTokens: $data['usage']['completion_tokens'] ?? 0,
            durationMs: $durationMs,
            stopReason: $data['choices'][0]['finish_reason'] ?? null,
            raw: $data,
        );
    }

    public function stream(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): Generator {
        $response = $this->client()
            ->withOptions(['stream' => true])
            ->post(self::API_URL, [
                'model' => $config['model'] ?? $this->model,
                'max_tokens' => $config['max_tokens'] ?? 4096,
                'temperature' => $config['temperature'] ?? 1.0,
                'stream' => true,
                'messages' => [
                    ['role' => 'system', 'content' => $systemPrompt],
                    ['role' => 'user', 'content' => $userPrompt],
                ],
            ]);

        yield from $this->parseSSEStream(
            $response->getBody(),
            fn (array $data) => $data['choices'][0]['delta']['content'] ?? null
        );
    }

    public function name(): string
    {
        return 'openai';
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
            'Authorization' => 'Bearer '.$this->apiKey,
            'Content-Type' => 'application/json',
        ])->timeout(300);
    }
}
