<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

interface AgenticProviderInterface
{
    /**
     * Generate a completion from the AI model.
     */
    public function generate(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): AgenticResponse;

    /**
     * Stream a completion from the AI model.
     *
     * @return Generator<string>
     */
    public function stream(
        string $systemPrompt,
        string $userPrompt,
        array $config = []
    ): \Generator;

    /**
     * Get the provider name.
     */
    public function name(): string;

    /**
     * Get the default model for this provider.
     */
    public function defaultModel(): string;

    /**
     * Check if the provider is configured and available.
     */
    public function isAvailable(): bool;
}
