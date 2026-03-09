<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Illuminate\Support\Facades\Log;
use InvalidArgumentException;

class AgenticManager
{
    /** @var array<string, AgenticProviderInterface> */
    private array $providers = [];

    private string $defaultProvider = 'claude';

    public function __construct()
    {
        $this->registerProviders();
    }

    /**
     * Get an AI provider by name.
     */
    public function provider(?string $name = null): AgenticProviderInterface
    {
        $name = $name ?? $this->defaultProvider;

        if (! isset($this->providers[$name])) {
            throw new InvalidArgumentException("Unknown AI provider: {$name}");
        }

        return $this->providers[$name];
    }

    /**
     * Get the Claude provider.
     */
    public function claude(): ClaudeService
    {
        return $this->providers['claude'];
    }

    /**
     * Get the Gemini provider.
     */
    public function gemini(): GeminiService
    {
        return $this->providers['gemini'];
    }

    /**
     * Get the OpenAI provider.
     */
    public function openai(): OpenAIService
    {
        return $this->providers['openai'];
    }

    /**
     * Get all available providers.
     *
     * @return array<string, AgenticProviderInterface>
     */
    public function availableProviders(): array
    {
        return array_filter(
            $this->providers,
            fn (AgenticProviderInterface $provider) => $provider->isAvailable()
        );
    }

    /**
     * Check if a provider is available.
     */
    public function isAvailable(string $name): bool
    {
        return isset($this->providers[$name]) && $this->providers[$name]->isAvailable();
    }

    /**
     * Set the default provider.
     */
    public function setDefault(string $name): void
    {
        if (! isset($this->providers[$name])) {
            throw new InvalidArgumentException("Unknown AI provider: {$name}");
        }

        $this->defaultProvider = $name;
    }

    /**
     * Register all AI providers.
     *
     * Logs a warning for each provider whose API key is absent so that
     * misconfiguration is surfaced at boot time rather than on the first
     * API call.  Set the corresponding environment variable to silence it:
     *
     *   ANTHROPIC_API_KEY  – Claude
     *   GOOGLE_AI_API_KEY  – Gemini
     *   OPENAI_API_KEY     – OpenAI
     */
    private function registerProviders(): void
    {
        // Use null coalescing since config() returns null for missing env vars
        $claudeKey = config('services.anthropic.api_key') ?? '';
        $geminiKey = config('services.google.ai_api_key') ?? '';
        $openaiKey = config('services.openai.api_key') ?? '';

        if (empty($claudeKey)) {
            Log::warning("Agentic: 'claude' provider has no API key configured. Set ANTHROPIC_API_KEY to enable it.");
        }

        if (empty($geminiKey)) {
            Log::warning("Agentic: 'gemini' provider has no API key configured. Set GOOGLE_AI_API_KEY to enable it.");
        }

        if (empty($openaiKey)) {
            Log::warning("Agentic: 'openai' provider has no API key configured. Set OPENAI_API_KEY to enable it.");
        }

        $this->providers['claude'] = new ClaudeService(
            apiKey: $claudeKey,
            model: config('services.anthropic.model') ?? 'claude-sonnet-4-20250514',
        );

        $this->providers['gemini'] = new GeminiService(
            apiKey: $geminiKey,
            model: config('services.google.ai_model') ?? 'gemini-2.0-flash',
        );

        $this->providers['openai'] = new OpenAIService(
            apiKey: $openaiKey,
            model: config('services.openai.model') ?? 'gpt-4o-mini',
        );
    }
}
