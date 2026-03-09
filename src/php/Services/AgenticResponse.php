<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

class AgenticResponse
{
    public function __construct(
        public readonly string $content,
        public readonly string $model,
        public readonly int $inputTokens,
        public readonly int $outputTokens,
        public readonly int $durationMs,
        public readonly ?string $stopReason = null,
        public readonly array $raw = [],
    ) {}

    /**
     * Get total tokens used.
     */
    public function totalTokens(): int
    {
        return $this->inputTokens + $this->outputTokens;
    }

    /**
     * Estimate cost based on model pricing.
     */
    public function estimateCost(): float
    {
        // Pricing per 1M tokens (approximate, as of Jan 2026)
        $pricing = [
            // Anthropic Claude models
            'claude-sonnet-4-20250514' => ['input' => 3.00, 'output' => 15.00],
            'claude-opus-4-20250514' => ['input' => 15.00, 'output' => 75.00],
            'claude-3-5-sonnet-20241022' => ['input' => 3.00, 'output' => 15.00],
            'claude-3-5-haiku-20241022' => ['input' => 0.80, 'output' => 4.00],

            // OpenAI GPT models
            'gpt-4o' => ['input' => 2.50, 'output' => 10.00],
            'gpt-4o-mini' => ['input' => 0.15, 'output' => 0.60],
            'gpt-4-turbo' => ['input' => 10.00, 'output' => 30.00],
            'gpt-4' => ['input' => 30.00, 'output' => 60.00],
            'gpt-3.5-turbo' => ['input' => 0.50, 'output' => 1.50],
            'o1' => ['input' => 15.00, 'output' => 60.00],
            'o1-mini' => ['input' => 3.00, 'output' => 12.00],
            'o1-preview' => ['input' => 15.00, 'output' => 60.00],

            // Google Gemini models
            'gemini-2.0-flash' => ['input' => 0.075, 'output' => 0.30],
            'gemini-2.0-flash-thinking' => ['input' => 0.70, 'output' => 3.50],
            'gemini-1.5-pro' => ['input' => 1.25, 'output' => 5.00],
            'gemini-1.5-flash' => ['input' => 0.075, 'output' => 0.30],
        ];

        $modelPricing = $pricing[$this->model] ?? ['input' => 0, 'output' => 0];

        return ($this->inputTokens * $modelPricing['input'] / 1_000_000) +
               ($this->outputTokens * $modelPricing['output'] / 1_000_000);
    }

    /**
     * Create from array.
     */
    public static function fromArray(array $data): self
    {
        return new self(
            content: $data['content'] ?? '',
            model: $data['model'] ?? 'unknown',
            inputTokens: $data['input_tokens'] ?? 0,
            outputTokens: $data['output_tokens'] ?? 0,
            durationMs: $data['duration_ms'] ?? 0,
            stopReason: $data['stop_reason'] ?? null,
            raw: $data['raw'] ?? [],
        );
    }
}
