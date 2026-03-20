<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Services\Concerns\HasRetry;
use Core\Mod\Agentic\Services\Concerns\HasStreamParsing;
use Generator;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;

class GeminiService implements AgenticProviderInterface
{
    use HasRetry;
    use HasStreamParsing;

    private const API_URL = 'https://generativelanguage.googleapis.com/v1beta/models';

    public function __construct(
        protected string $apiKey,
        protected string $model = 'gemini-2.0-flash',
    ) {}

    public function generate(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): AgenticResponse {
        $startTime = microtime(true);
        $model = $config['model'] ?? $this->model;

        $response = $this->withRetry(
            fn () => $this->client()->post(
                self::API_URL."/{$model}:generateContent",
                [
                    'contents' => [
                        [
                            'parts' => [
                                ['text' => $userPrompt],
                            ],
                        ],
                    ],
                    'systemInstruction' => [
                        'parts' => [
                            ['text' => $systemPrompt],
                        ],
                    ],
                    'generationConfig' => [
                        'temperature' => $config['temperature'] ?? 1.0,
                        'maxOutputTokens' => $config['max_tokens'] ?? 4096,
                    ],
                ]
            ),
            'Gemini'
        );

        $data = $response->json();
        $durationMs = (int) ((microtime(true) - $startTime) * 1000);

        $content = $data['candidates'][0]['content']['parts'][0]['text'] ?? '';
        $usageMetadata = $data['usageMetadata'] ?? [];

        return new AgenticResponse(
            content: $content,
            model: $model,
            inputTokens: $usageMetadata['promptTokenCount'] ?? 0,
            outputTokens: $usageMetadata['candidatesTokenCount'] ?? 0,
            durationMs: $durationMs,
            stopReason: $data['candidates'][0]['finishReason'] ?? null,
            raw: $data,
        );
    }

    public function stream(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): Generator {
        $model = $config['model'] ?? $this->model;

        $response = $this->client()
            ->withOptions(['stream' => true])
            ->post(
                self::API_URL."/{$model}:streamGenerateContent",
                [
                    'contents' => [
                        [
                            'parts' => [
                                ['text' => $userPrompt],
                            ],
                        ],
                    ],
                    'systemInstruction' => [
                        'parts' => [
                            ['text' => $systemPrompt],
                        ],
                    ],
                    'generationConfig' => [
                        'temperature' => $config['temperature'] ?? 1.0,
                        'maxOutputTokens' => $config['max_tokens'] ?? 4096,
                    ],
                ]
            );

        // Gemini uses JSON array streaming, not SSE
        yield from $this->parseJSONStream(
            $response->getBody(),
            fn (array $data) => $data['candidates'][0]['content']['parts'][0]['text'] ?? null
        );
    }

    public function name(): string
    {
        return 'gemini';
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
            'Content-Type' => 'application/json',
        ])->withQueryParameters([
            'key' => $this->apiKey,
        ])->timeout(300);
    }
}
