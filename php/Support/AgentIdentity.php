<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Support;

/**
 * Represents the identity of an AI agent making a request.
 *
 * Used by AgentDetection service to identify AI providers from User-Agent
 * strings and MCP tokens. Part of the Trees for Agents system.
 */
class AgentIdentity
{
    /**
     * Confidence levels for agent detection.
     */
    public const CONFIDENCE_HIGH = 'high';

    public const CONFIDENCE_MEDIUM = 'medium';

    public const CONFIDENCE_LOW = 'low';

    public function __construct(
        public readonly string $provider,
        public readonly ?string $model,
        public readonly string $confidence
    ) {}

    /**
     * Check if this represents an AI agent (not a regular user).
     */
    public function isAgent(): bool
    {
        return $this->provider !== 'not_agent';
    }

    /**
     * Check if this is not an AI agent (regular user).
     */
    public function isNotAgent(): bool
    {
        return ! $this->isAgent();
    }

    /**
     * Check if this is a known agent (not unknown).
     */
    public function isKnown(): bool
    {
        return $this->isAgent() && $this->provider !== 'unknown';
    }

    /**
     * Check if this is an unknown agent.
     */
    public function isUnknown(): bool
    {
        return $this->provider === 'unknown';
    }

    /**
     * Check if detection confidence is high.
     */
    public function isHighConfidence(): bool
    {
        return $this->confidence === self::CONFIDENCE_HIGH;
    }

    /**
     * Check if detection confidence is medium or higher.
     */
    public function isMediumConfidenceOrHigher(): bool
    {
        return in_array($this->confidence, [self::CONFIDENCE_HIGH, self::CONFIDENCE_MEDIUM], true);
    }

    /**
     * Get the referral URL path for this agent.
     *
     * @return string|null URL path like "/ref/anthropic/claude-opus" or null if not an agent
     */
    public function getReferralPath(): ?string
    {
        if ($this->isNotAgent()) {
            return null;
        }

        if ($this->model) {
            return "/ref/{$this->provider}/{$this->model}";
        }

        return "/ref/{$this->provider}";
    }

    /**
     * Create an identity representing a regular user (not an agent).
     */
    public static function notAnAgent(): self
    {
        return new self('not_agent', null, self::CONFIDENCE_HIGH);
    }

    /**
     * Create an identity for an unknown agent.
     *
     * Used when we detect programmatic access but can't identify the provider.
     */
    public static function unknownAgent(): self
    {
        return new self('unknown', null, self::CONFIDENCE_LOW);
    }

    /**
     * Create an identity for Anthropic/Claude.
     */
    public static function anthropic(?string $model = null, string $confidence = self::CONFIDENCE_HIGH): self
    {
        return new self('anthropic', $model, $confidence);
    }

    /**
     * Create an identity for OpenAI/ChatGPT.
     */
    public static function openai(?string $model = null, string $confidence = self::CONFIDENCE_HIGH): self
    {
        return new self('openai', $model, $confidence);
    }

    /**
     * Create an identity for Google/Gemini.
     */
    public static function google(?string $model = null, string $confidence = self::CONFIDENCE_HIGH): self
    {
        return new self('google', $model, $confidence);
    }

    /**
     * Create an identity for Meta/LLaMA.
     */
    public static function meta(?string $model = null, string $confidence = self::CONFIDENCE_HIGH): self
    {
        return new self('meta', $model, $confidence);
    }

    /**
     * Create an identity for Mistral.
     */
    public static function mistral(?string $model = null, string $confidence = self::CONFIDENCE_HIGH): self
    {
        return new self('mistral', $model, $confidence);
    }

    /**
     * Create an identity for local/self-hosted models.
     */
    public static function local(?string $model = null, string $confidence = self::CONFIDENCE_MEDIUM): self
    {
        return new self('local', $model, $confidence);
    }

    /**
     * Get the provider display name.
     */
    public function getProviderDisplayName(): string
    {
        return match ($this->provider) {
            'anthropic' => 'Anthropic',
            'openai' => 'OpenAI',
            'google' => 'Google',
            'meta' => 'Meta',
            'mistral' => 'Mistral',
            'local' => 'Local Model',
            'unknown' => 'Unknown Agent',
            'not_agent' => 'User',
            default => ucfirst($this->provider),
        };
    }

    /**
     * Get the model display name.
     */
    public function getModelDisplayName(): ?string
    {
        if (! $this->model) {
            return null;
        }

        // Normalise common model names for display
        return match (strtolower($this->model)) {
            'claude-opus', 'claude-opus-4' => 'Claude Opus',
            'claude-sonnet', 'claude-sonnet-4' => 'Claude Sonnet',
            'claude-haiku', 'claude-haiku-3' => 'Claude Haiku',
            'gpt-4', 'gpt-4o', 'gpt-4-turbo' => 'GPT-4',
            'gpt-3.5', 'gpt-3.5-turbo' => 'GPT-3.5',
            'o1', 'o1-preview', 'o1-mini' => 'o1',
            'gemini-pro', 'gemini-1.5-pro' => 'Gemini Pro',
            'gemini-ultra', 'gemini-1.5-ultra' => 'Gemini Ultra',
            'gemini-flash', 'gemini-1.5-flash' => 'Gemini Flash',
            'llama-3', 'llama-3.1', 'llama-3.2' => 'LLaMA 3',
            'mistral-large', 'mistral-medium', 'mistral-small' => ucfirst($this->model),
            default => $this->model,
        };
    }

    /**
     * Convert to array for API responses.
     */
    public function toArray(): array
    {
        return [
            'provider' => $this->provider,
            'model' => $this->model,
            'confidence' => $this->confidence,
            'is_agent' => $this->isAgent(),
            'referral_path' => $this->getReferralPath(),
        ];
    }
}
