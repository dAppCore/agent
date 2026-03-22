<?php

declare(strict_types=1);

/**
 * Tests for the AgenticManager AI provider coordinator.
 *
 * Covers provider registration, retrieval, availability checks, and default provider handling.
 * Uses mocked configuration to test the manager without real API keys.
 */

use Core\Mod\Agentic\Services\AgenticManager;
use Core\Mod\Agentic\Services\AgenticProviderInterface;
use Core\Mod\Agentic\Services\ClaudeService;
use Core\Mod\Agentic\Services\GeminiService;
use Core\Mod\Agentic\Services\OpenAIService;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\Log;
use InvalidArgumentException;

// =========================================================================
// Provider Registration Tests
// =========================================================================

describe('provider registration', function () {
    it('registers all three providers on construction', function () {
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');

        $manager = new AgenticManager;

        expect($manager->claude())->toBeInstanceOf(ClaudeService::class)
            ->and($manager->gemini())->toBeInstanceOf(GeminiService::class)
            ->and($manager->openai())->toBeInstanceOf(OpenAIService::class);
    });

    it('uses configured model for Claude provider', function () {
        Config::set('services.anthropic.api_key', 'test-key');
        Config::set('services.anthropic.model', 'claude-opus-4-20250514');

        $manager = new AgenticManager;

        expect($manager->claude()->defaultModel())->toBe('claude-opus-4-20250514');
    });

    it('uses configured model for Gemini provider', function () {
        Config::set('services.google.ai_api_key', 'test-key');
        Config::set('services.google.ai_model', 'gemini-1.5-pro');

        $manager = new AgenticManager;

        expect($manager->gemini()->defaultModel())->toBe('gemini-1.5-pro');
    });

    it('uses configured model for OpenAI provider', function () {
        Config::set('services.openai.api_key', 'test-key');
        Config::set('services.openai.model', 'gpt-4o');

        $manager = new AgenticManager;

        expect($manager->openai()->defaultModel())->toBe('gpt-4o');
    });

    it('uses default model when not configured for Claude', function () {
        Config::set('services.anthropic.api_key', 'test-key');
        Config::set('services.anthropic.model', null);

        $manager = new AgenticManager;

        expect($manager->claude()->defaultModel())->toBe('claude-sonnet-4-20250514');
    });

    it('uses default model when not configured for Gemini', function () {
        Config::set('services.google.ai_api_key', 'test-key');
        Config::set('services.google.ai_model', null);

        $manager = new AgenticManager;

        expect($manager->gemini()->defaultModel())->toBe('gemini-2.0-flash');
    });

    it('uses default model when not configured for OpenAI', function () {
        Config::set('services.openai.api_key', 'test-key');
        Config::set('services.openai.model', null);

        $manager = new AgenticManager;

        expect($manager->openai()->defaultModel())->toBe('gpt-4o-mini');
    });
});

// =========================================================================
// Provider Retrieval Tests
// =========================================================================

describe('provider retrieval', function () {
    beforeEach(function () {
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');
    });

    it('retrieves provider by name using provider() method', function () {
        $manager = new AgenticManager;

        expect($manager->provider('claude'))->toBeInstanceOf(ClaudeService::class)
            ->and($manager->provider('gemini'))->toBeInstanceOf(GeminiService::class)
            ->and($manager->provider('openai'))->toBeInstanceOf(OpenAIService::class);
    });

    it('returns default provider when null passed to provider()', function () {
        $manager = new AgenticManager;

        // Default is 'claude'
        expect($manager->provider(null))->toBeInstanceOf(ClaudeService::class);
    });

    it('returns default provider when no argument passed to provider()', function () {
        $manager = new AgenticManager;

        expect($manager->provider())->toBeInstanceOf(ClaudeService::class);
    });

    it('throws exception for unknown provider name', function () {
        $manager = new AgenticManager;

        expect(fn () => $manager->provider('unknown'))
            ->toThrow(InvalidArgumentException::class, 'Unknown AI provider: unknown');
    });

    it('returns provider implementing AgenticProviderInterface', function () {
        $manager = new AgenticManager;

        expect($manager->provider('claude'))->toBeInstanceOf(AgenticProviderInterface::class);
    });
});

// =========================================================================
// Default Provider Tests
// =========================================================================

describe('default provider', function () {
    beforeEach(function () {
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');
    });

    it('uses claude as default provider initially', function () {
        $manager = new AgenticManager;

        expect($manager->provider()->name())->toBe('claude');
    });

    it('allows changing default provider to gemini', function () {
        $manager = new AgenticManager;

        $manager->setDefault('gemini');

        expect($manager->provider()->name())->toBe('gemini');
    });

    it('allows changing default provider to openai', function () {
        $manager = new AgenticManager;

        $manager->setDefault('openai');

        expect($manager->provider()->name())->toBe('openai');
    });

    it('throws exception when setting unknown default provider', function () {
        $manager = new AgenticManager;

        expect(fn () => $manager->setDefault('unknown'))
            ->toThrow(InvalidArgumentException::class, 'Unknown AI provider: unknown');
    });

    it('allows switching default provider multiple times', function () {
        $manager = new AgenticManager;

        $manager->setDefault('gemini');
        expect($manager->provider()->name())->toBe('gemini');

        $manager->setDefault('openai');
        expect($manager->provider()->name())->toBe('openai');

        $manager->setDefault('claude');
        expect($manager->provider()->name())->toBe('claude');
    });
});

// =========================================================================
// Provider Availability Tests
// =========================================================================

describe('provider availability', function () {
    it('reports provider as available when API key is set', function () {
        Config::set('services.anthropic.api_key', 'test-key');

        $manager = new AgenticManager;

        expect($manager->isAvailable('claude'))->toBeTrue();
    });

    it('reports provider as unavailable when API key is empty', function () {
        Config::set('services.anthropic.api_key', '');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', '');

        $manager = new AgenticManager;

        expect($manager->isAvailable('claude'))->toBeFalse()
            ->and($manager->isAvailable('gemini'))->toBeFalse()
            ->and($manager->isAvailable('openai'))->toBeFalse();
    });

    it('reports provider as unavailable when API key is null', function () {
        Config::set('services.anthropic.api_key', null);

        $manager = new AgenticManager;

        expect($manager->isAvailable('claude'))->toBeFalse();
    });

    it('returns false for unknown provider name', function () {
        $manager = new AgenticManager;

        expect($manager->isAvailable('unknown'))->toBeFalse();
    });

    it('checks availability independently for each provider', function () {
        Config::set('services.anthropic.api_key', 'test-key');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', 'test-key');

        $manager = new AgenticManager;

        expect($manager->isAvailable('claude'))->toBeTrue()
            ->and($manager->isAvailable('gemini'))->toBeFalse()
            ->and($manager->isAvailable('openai'))->toBeTrue();
    });
});

// =========================================================================
// Available Providers Tests
// =========================================================================

describe('available providers list', function () {
    it('returns all providers when all have API keys', function () {
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');

        $manager = new AgenticManager;
        $available = $manager->availableProviders();

        expect($available)->toHaveCount(3)
            ->and(array_keys($available))->toBe(['claude', 'gemini', 'openai']);
    });

    it('returns empty array when no providers have API keys', function () {
        Config::set('services.anthropic.api_key', '');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', '');

        $manager = new AgenticManager;

        expect($manager->availableProviders())->toBeEmpty();
    });

    it('returns only providers with valid API keys', function () {
        Config::set('services.anthropic.api_key', 'test-key');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', 'test-key');

        $manager = new AgenticManager;
        $available = $manager->availableProviders();

        expect($available)->toHaveCount(2)
            ->and(array_keys($available))->toBe(['claude', 'openai']);
    });

    it('returns providers implementing AgenticProviderInterface', function () {
        Config::set('services.anthropic.api_key', 'test-key');

        $manager = new AgenticManager;
        $available = $manager->availableProviders();

        foreach ($available as $provider) {
            expect($provider)->toBeInstanceOf(AgenticProviderInterface::class);
        }
    });
});

// =========================================================================
// Direct Provider Access Tests
// =========================================================================

describe('direct provider access methods', function () {
    beforeEach(function () {
        Config::set('services.anthropic.api_key', 'test-key');
        Config::set('services.google.ai_api_key', 'test-key');
        Config::set('services.openai.api_key', 'test-key');
    });

    it('returns ClaudeService from claude() method', function () {
        $manager = new AgenticManager;

        expect($manager->claude())
            ->toBeInstanceOf(ClaudeService::class)
            ->and($manager->claude()->name())->toBe('claude');
    });

    it('returns GeminiService from gemini() method', function () {
        $manager = new AgenticManager;

        expect($manager->gemini())
            ->toBeInstanceOf(GeminiService::class)
            ->and($manager->gemini()->name())->toBe('gemini');
    });

    it('returns OpenAIService from openai() method', function () {
        $manager = new AgenticManager;

        expect($manager->openai())
            ->toBeInstanceOf(OpenAIService::class)
            ->and($manager->openai()->name())->toBe('openai');
    });

    it('returns same instance on repeated calls', function () {
        $manager = new AgenticManager;

        $claude1 = $manager->claude();
        $claude2 = $manager->claude();

        expect($claude1)->toBe($claude2);
    });
});

// =========================================================================
// Edge Case Tests
// =========================================================================

describe('edge cases', function () {
    it('handles missing configuration gracefully', function () {
        Log::spy();

        Config::set('services.anthropic.api_key', null);
        Config::set('services.anthropic.model', null);
        Config::set('services.google.ai_api_key', null);
        Config::set('services.google.ai_model', null);
        Config::set('services.openai.api_key', null);
        Config::set('services.openai.model', null);

        $manager = new AgenticManager;

        // Should still construct without throwing
        expect($manager->claude())->toBeInstanceOf(ClaudeService::class)
            ->and($manager->gemini())->toBeInstanceOf(GeminiService::class)
            ->and($manager->openai())->toBeInstanceOf(OpenAIService::class);

        // But all should be unavailable
        expect($manager->availableProviders())->toBeEmpty();

        // Warnings logged for all three unconfigured providers
        Log::shouldHaveReceived('warning')->times(3);
    });

    it('provider retrieval is case-sensitive', function () {
        Config::set('services.anthropic.api_key', 'test-key');

        $manager = new AgenticManager;

        expect(fn () => $manager->provider('Claude'))
            ->toThrow(InvalidArgumentException::class);
    });

    it('isAvailable handles case sensitivity', function () {
        Config::set('services.anthropic.api_key', 'test-key');

        $manager = new AgenticManager;

        expect($manager->isAvailable('claude'))->toBeTrue()
            ->and($manager->isAvailable('Claude'))->toBeFalse()
            ->and($manager->isAvailable('CLAUDE'))->toBeFalse();
    });

    it('setDefault handles case sensitivity', function () {
        $manager = new AgenticManager;

        expect(fn () => $manager->setDefault('Gemini'))
            ->toThrow(InvalidArgumentException::class);
    });
});

// =========================================================================
// API Key Validation Warning Tests
// =========================================================================

describe('API key validation warnings', function () {
    it('logs a warning when Claude API key is not configured', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', '');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');

        new AgenticManager;

        Log::shouldHaveReceived('warning')
            ->once()
            ->withArgs(fn (string $message) => str_contains($message, 'claude') && str_contains($message, 'ANTHROPIC_API_KEY'));
    });

    it('logs a warning when Gemini API key is not configured', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', 'test-openai-key');

        new AgenticManager;

        Log::shouldHaveReceived('warning')
            ->once()
            ->withArgs(fn (string $message) => str_contains($message, 'gemini') && str_contains($message, 'GOOGLE_AI_API_KEY'));
    });

    it('logs a warning when OpenAI API key is not configured', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', '');

        new AgenticManager;

        Log::shouldHaveReceived('warning')
            ->once()
            ->withArgs(fn (string $message) => str_contains($message, 'openai') && str_contains($message, 'OPENAI_API_KEY'));
    });

    it('logs a warning when API key is null', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', null);
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');

        new AgenticManager;

        Log::shouldHaveReceived('warning')
            ->once()
            ->withArgs(fn (string $message) => str_contains($message, 'claude'));
    });

    it('logs warnings for all three providers when no keys are configured', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', '');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', '');

        new AgenticManager;

        Log::shouldHaveReceived('warning')->times(3);
    });

    it('does not log warnings when all API keys are configured', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', 'test-claude-key');
        Config::set('services.google.ai_api_key', 'test-gemini-key');
        Config::set('services.openai.api_key', 'test-openai-key');

        new AgenticManager;

        Log::shouldNotHaveReceived('warning');
    });

    it('only warns for providers that have missing keys, not all providers', function () {
        Log::spy();
        Config::set('services.anthropic.api_key', 'test-key');
        Config::set('services.google.ai_api_key', '');
        Config::set('services.openai.api_key', '');

        new AgenticManager;

        // Only gemini and openai should warn – not claude
        Log::shouldHaveReceived('warning')->times(2);
    });
});
