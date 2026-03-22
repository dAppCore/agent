<?php

declare(strict_types=1);

/**
 * Tests for the AgentDetection service.
 *
 * Covers User-Agent pattern matching for known AI providers, browser and
 * non-agent bot detection, MCP token identification, and edge cases.
 * Documents the UA patterns used to identify each agent type.
 *
 * Resolves: #13
 */

use Core\Mod\Agentic\Services\AgentDetection;
use Core\Mod\Agentic\Support\AgentIdentity;
use Illuminate\Http\Request;

// =========================================================================
// Edge Cases
// =========================================================================

describe('edge cases', function () {
    it('returns unknownAgent for null User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(null);

        expect($identity->provider)->toBe('unknown')
            ->and($identity->isAgent())->toBeTrue()
            ->and($identity->isKnown())->toBeFalse()
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_LOW);
    });

    it('returns unknownAgent for empty string User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('');

        expect($identity->provider)->toBe('unknown')
            ->and($identity->isAgent())->toBeTrue();
    });

    it('returns unknownAgent for whitespace-only User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('   ');

        expect($identity->provider)->toBe('unknown')
            ->and($identity->isAgent())->toBeTrue();
    });

    it('returns unknownAgent for generic programmatic client with no known indicators', function () {
        $service = new AgentDetection;
        // A plain HTTP client string without browser or bot keywords
        $identity = $service->identifyFromUserAgent('my-custom-client/1.0');

        expect($identity->provider)->toBe('unknown')
            ->and($identity->isAgent())->toBeTrue()
            ->and($identity->isKnown())->toBeFalse();
    });

    it('returns unknownAgent for numeric-only User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('1.0');

        expect($identity->provider)->toBe('unknown');
    });
});

// =========================================================================
// Anthropic / Claude Detection
// =========================================================================

describe('Anthropic/Claude detection', function () {
    /**
     * Pattern: /claude[\s\-_]?code/i
     * Examples: "claude-code/1.2.3", "ClaudeCode/1.0", "claude_code"
     */
    it('detects Claude Code User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('claude-code/1.2.3');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->isKnown())->toBeTrue()
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Pattern: /\banthropic[\s\-_]?api\b/i
     * Examples: "anthropic-api/1.0", "Anthropic API Client/2.0"
     */
    it('detects Anthropic API User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('anthropic-api/1.0 Python/3.11');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Pattern: /\bclaude\b.*\bai\b/i
     * Examples: "Claude AI/2.0", "claude ai client"
     */
    it('detects Claude AI User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Claude AI Agent/1.0');

        expect($identity->provider)->toBe('anthropic');
    });

    /**
     * Pattern: /\bclaude\b.*\bassistant\b/i
     * Examples: "claude assistant/1.0", "Claude Assistant integration"
     */
    it('detects Claude Assistant User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('claude assistant integration/2.0');

        expect($identity->provider)->toBe('anthropic');
    });

    /**
     * Model pattern: /claude[\s\-_]?opus/i
     * Examples: "claude-opus", "Claude Opus", "claude_opus"
     */
    it('detects claude-opus model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('claude-opus claude-code/1.0');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->model)->toBe('claude-opus');
    });

    /**
     * Model pattern: /claude[\s\-_]?sonnet/i
     * Examples: "claude-sonnet", "Claude Sonnet", "claude_sonnet"
     */
    it('detects claude-sonnet model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('claude-sonnet claude-code/1.0');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->model)->toBe('claude-sonnet');
    });

    /**
     * Model pattern: /claude[\s\-_]?haiku/i
     * Examples: "claude-haiku", "Claude Haiku", "claude_haiku"
     */
    it('detects claude-haiku model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Claude Haiku claude-code/1.0');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->model)->toBe('claude-haiku');
    });

    it('returns null model when no Anthropic model pattern matches', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('claude-code/1.0');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->model)->toBeNull();
    });
});

// =========================================================================
// OpenAI / ChatGPT Detection
// =========================================================================

describe('OpenAI/ChatGPT detection', function () {
    /**
     * Pattern: /\bChatGPT\b/i
     * Examples: "ChatGPT/1.2", "chatgpt-plugin/1.0"
     */
    it('detects ChatGPT User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('ChatGPT/1.2 OpenAI');

        expect($identity->provider)->toBe('openai')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Pattern: /\bOpenAI\b/i
     * Examples: "OpenAI Python SDK/1.0", "openai-node/4.0"
     */
    it('detects OpenAI User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('OpenAI Python SDK/1.0');

        expect($identity->provider)->toBe('openai');
    });

    /**
     * Pattern: /\bGPT[\s\-_]?4\b/i
     * Model pattern: /\bGPT[\s\-_]?4/i
     * Examples: "GPT-4 Agent/1.0", "GPT4 client", "GPT 4"
     */
    it('detects GPT-4 and sets gpt-4 model', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('GPT-4 Agent/1.0');

        expect($identity->provider)->toBe('openai')
            ->and($identity->model)->toBe('gpt-4');
    });

    /**
     * Pattern: /\bGPT[\s\-_]?3\.?5\b/i
     * Model pattern: /\bGPT[\s\-_]?3\.?5/i
     * Examples: "GPT-3.5 Turbo", "GPT35 client", "GPT 3.5"
     */
    it('detects GPT-3.5 and sets gpt-3.5 model', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('GPT-3.5 Turbo client/1.0');

        expect($identity->provider)->toBe('openai')
            ->and($identity->model)->toBe('gpt-3.5');
    });

    /**
     * Pattern: /\bo1[\s\-_]?preview\b/i
     * Examples: "o1-preview OpenAI client/1.0"
     */
    it('detects o1-preview User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('o1-preview OpenAI client/1.0');

        expect($identity->provider)->toBe('openai');
    });

    /**
     * Pattern: /\bo1[\s\-_]?mini\b/i
     * Examples: "o1-mini OpenAI client/1.0"
     */
    it('detects o1-mini User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('o1-mini OpenAI client/1.0');

        expect($identity->provider)->toBe('openai');
    });
});

// =========================================================================
// Google / Gemini Detection
// =========================================================================

describe('Google/Gemini detection', function () {
    /**
     * Pattern: /\bGoogle[\s\-_]?AI\b/i
     * Examples: "Google AI Studio/1.0", "GoogleAI/2.0"
     */
    it('detects Google AI User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Google AI Studio/1.0');

        expect($identity->provider)->toBe('google')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Pattern: /\bGemini\b/i
     * Examples: "Gemini API Client/2.0", "gemini-client/1.0"
     */
    it('detects Gemini User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Gemini API Client/2.0');

        expect($identity->provider)->toBe('google');
    });

    /**
     * Pattern: /\bBard\b/i
     * Examples: "Bard/1.0", "Google Bard client"
     */
    it('detects Bard User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Bard/1.0');

        expect($identity->provider)->toBe('google');
    });

    /**
     * Pattern: /\bPaLM\b/i
     * Examples: "PaLM API/2.0", "Google PaLM"
     */
    it('detects PaLM User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('PaLM API/2.0');

        expect($identity->provider)->toBe('google');
    });

    /**
     * Model pattern: /gemini[\s\-_]?(1\.5[\s\-_]?)?pro/i
     * Examples: "Gemini Pro client/1.0", "gemini-pro/1.0", "gemini-1.5-pro"
     */
    it('detects gemini-pro model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Gemini Pro client/1.0');

        expect($identity->provider)->toBe('google')
            ->and($identity->model)->toBe('gemini-pro');
    });

    /**
     * Model pattern: /gemini[\s\-_]?(1\.5[\s\-_]?)?flash/i
     * Examples: "gemini-flash/1.5", "Gemini Flash client", "gemini-1.5-flash"
     */
    it('detects gemini-flash model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('gemini-flash/1.5');

        expect($identity->provider)->toBe('google')
            ->and($identity->model)->toBe('gemini-flash');
    });

    /**
     * Model pattern: /gemini[\s\-_]?(1\.5[\s\-_]?)?ultra/i
     * Examples: "Gemini Ultra/1.0", "gemini-1.5-ultra"
     */
    it('detects gemini-ultra model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Gemini Ultra/1.0');

        expect($identity->provider)->toBe('google')
            ->and($identity->model)->toBe('gemini-ultra');
    });
});

// =========================================================================
// Meta / LLaMA Detection
// =========================================================================

describe('Meta/LLaMA detection', function () {
    /**
     * Pattern: /\bMeta[\s\-_]?AI\b/i
     * Examples: "Meta AI assistant/1.0", "MetaAI/1.0"
     */
    it('detects Meta AI User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Meta AI assistant/1.0');

        expect($identity->provider)->toBe('meta')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Pattern: /\bLLaMA\b/i
     * Examples: "LLaMA model client/1.0", "llama-inference"
     */
    it('detects LLaMA User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('LLaMA model client/1.0');

        expect($identity->provider)->toBe('meta');
    });

    /**
     * Pattern: /\bLlama[\s\-_]?[23]\b/i
     * Model pattern: /llama[\s\-_]?3/i
     * Examples: "Llama-3 inference client/1.0", "Llama3/1.0", "Llama 3"
     */
    it('detects Llama 3 and sets llama-3 model', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Llama-3 inference client/1.0');

        expect($identity->provider)->toBe('meta')
            ->and($identity->model)->toBe('llama-3');
    });

    /**
     * Pattern: /\bLlama[\s\-_]?[23]\b/i
     * Model pattern: /llama[\s\-_]?2/i
     * Examples: "Llama-2 inference client/1.0", "Llama2/1.0", "Llama 2"
     */
    it('detects Llama 2 and sets llama-2 model', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Llama-2 inference client/1.0');

        expect($identity->provider)->toBe('meta')
            ->and($identity->model)->toBe('llama-2');
    });
});

// =========================================================================
// Mistral Detection
// =========================================================================

describe('Mistral detection', function () {
    /**
     * Pattern: /\bMistral\b/i
     * Examples: "Mistral AI client/1.0", "mistral-python/1.0"
     */
    it('detects Mistral User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Mistral AI client/1.0');

        expect($identity->provider)->toBe('mistral')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Pattern: /\bMixtral\b/i
     * Model pattern: /mixtral/i
     * Examples: "Mixtral-8x7B client/1.0", "mixtral inference"
     */
    it('detects Mixtral User-Agent and sets mixtral model', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Mixtral-8x7B client/1.0');

        expect($identity->provider)->toBe('mistral')
            ->and($identity->model)->toBe('mixtral');
    });

    /**
     * Model pattern: /mistral[\s\-_]?large/i
     * Examples: "Mistral Large API/2.0", "mistral-large/1.0"
     */
    it('detects mistral-large model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Mistral Large API/2.0');

        expect($identity->provider)->toBe('mistral')
            ->and($identity->model)->toBe('mistral-large');
    });

    /**
     * Model pattern: /mistral[\s\-_]?medium/i
     * Examples: "Mistral Medium/1.0", "mistral-medium client"
     */
    it('detects mistral-medium model from User-Agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('mistral-medium client/1.0');

        expect($identity->provider)->toBe('mistral')
            ->and($identity->model)->toBe('mistral-medium');
    });
});

// =========================================================================
// Browser Detection (not an agent)
// =========================================================================

describe('browser detection', function () {
    /**
     * Pattern: /\bMozilla\b/i
     * All modern browsers include "Mozilla/5.0" in their UA string.
     * Chrome example: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ... Chrome/120..."
     */
    it('detects Chrome browser as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
        );

        expect($identity->isNotAgent())->toBeTrue()
            ->and($identity->provider)->toBe('not_agent')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Firefox example: "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0"
     */
    it('detects Firefox browser as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0'
        );

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Safari example: "Mozilla/5.0 (Macintosh; ...) ... Version/17.0 Safari/605.1.15"
     */
    it('detects Safari browser as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15'
        );

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Edge example: "Mozilla/5.0 ... Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"
     */
    it('detects Edge browser as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0'
        );

        expect($identity->isNotAgent())->toBeTrue();
    });
});

// =========================================================================
// Non-Agent Bot Detection
// =========================================================================

describe('non-agent bot detection', function () {
    /**
     * Pattern: /\bGooglebot\b/i
     * Example: "Googlebot/2.1 (+http://www.google.com/bot.html)"
     */
    it('detects Googlebot as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Googlebot/2.1 (+http://www.google.com/bot.html)'
        );

        expect($identity->isNotAgent())->toBeTrue()
            ->and($identity->provider)->toBe('not_agent');
    });

    /**
     * Pattern: /\bBingbot\b/i
     * Example: "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)"
     */
    it('detects Bingbot as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)'
        );

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\bcurl\b/i
     * Example: "curl/7.68.0"
     */
    it('detects curl as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('curl/7.68.0');

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\bpython-requests\b/i
     * Example: "python-requests/2.28.0"
     */
    it('detects python-requests as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('python-requests/2.28.0');

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\bPostman\b/i
     * Example: "PostmanRuntime/7.32.0"
     */
    it('detects Postman as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('PostmanRuntime/7.32.0');

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\bSlackbot\b/i
     * Example: "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)"
     */
    it('detects Slackbot as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent(
            'Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)'
        );

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\bgo-http-client\b/i
     * Example: "Go-http-client/1.1"
     */
    it('detects Go-http-client as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('Go-http-client/1.1');

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\baxios\b/i
     * Example: "axios/1.4.0"
     */
    it('detects axios as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('axios/1.4.0');

        expect($identity->isNotAgent())->toBeTrue();
    });

    /**
     * Pattern: /\bnode-fetch\b/i
     * Example: "node-fetch/2.6.9"
     */
    it('detects node-fetch as not an agent', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromUserAgent('node-fetch/2.6.9');

        expect($identity->isNotAgent())->toBeTrue();
    });
});

// =========================================================================
// MCP Token Detection
// =========================================================================

describe('MCP token detection', function () {
    /**
     * Structured token format: "provider:model:secret"
     * Example: "anthropic:claude-opus:abc123"
     */
    it('identifies Anthropic from structured MCP token', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromMcpToken('anthropic:claude-opus:secret123');

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->model)->toBe('claude-opus')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Structured token format: "provider:model:secret"
     * Example: "openai:gpt-4:xyz789"
     */
    it('identifies OpenAI from structured MCP token', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromMcpToken('openai:gpt-4:secret456');

        expect($identity->provider)->toBe('openai')
            ->and($identity->model)->toBe('gpt-4')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    /**
     * Structured token format: "provider:model:secret"
     * Example: "google:gemini-pro:zyx321"
     */
    it('identifies Google from structured MCP token', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromMcpToken('google:gemini-pro:secret789');

        expect($identity->provider)->toBe('google')
            ->and($identity->model)->toBe('gemini-pro')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_HIGH);
    });

    it('accepts meta and mistral providers in structured tokens', function () {
        $service = new AgentDetection;

        expect($service->identifyFromMcpToken('meta:llama-3:secret')->provider)->toBe('meta');
        expect($service->identifyFromMcpToken('mistral:mistral-large:secret')->provider)->toBe('mistral');
    });

    it('returns medium-confidence unknown for unrecognised token string', function () {
        $service = new AgentDetection;
        // No colon separator — cannot parse as structured token
        $identity = $service->identifyFromMcpToken('some-random-opaque-token');

        expect($identity->provider)->toBe('unknown')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_MEDIUM);
    });

    it('returns medium-confidence unknown for structured token with invalid provider', function () {
        $service = new AgentDetection;
        $identity = $service->identifyFromMcpToken('facebook:llama:secret');

        expect($identity->provider)->toBe('unknown')
            ->and($identity->confidence)->toBe(AgentIdentity::CONFIDENCE_MEDIUM);
    });

    it('prioritises MCP token header over User-Agent in HTTP request', function () {
        $service = new AgentDetection;
        $request = Request::create('/test', 'GET', [], [], [], [
            'HTTP_X_MCP_TOKEN' => 'anthropic:claude-sonnet:token123',
            'HTTP_USER_AGENT' => 'python-requests/2.28.0',
        ]);

        // MCP token takes precedence; UA would indicate notAnAgent otherwise
        $identity = $service->identify($request);

        expect($identity->provider)->toBe('anthropic')
            ->and($identity->model)->toBe('claude-sonnet');
    });

    it('falls back to User-Agent when no MCP token header is present', function () {
        $service = new AgentDetection;
        $request = Request::create('/test', 'GET', [], [], [], [
            'HTTP_USER_AGENT' => 'claude-code/1.0',
        ]);

        $identity = $service->identify($request);

        expect($identity->provider)->toBe('anthropic');
    });
});

// =========================================================================
// Provider Validation
// =========================================================================

describe('provider validation', function () {
    it('accepts all known valid providers', function () {
        $service = new AgentDetection;
        $validProviders = ['anthropic', 'openai', 'google', 'meta', 'mistral', 'local', 'unknown'];

        foreach ($validProviders as $provider) {
            expect($service->isValidProvider($provider))
                ->toBeTrue("Expected '{$provider}' to be a valid provider");
        }
    });

    it('rejects unknown provider names', function () {
        $service = new AgentDetection;

        expect($service->isValidProvider('facebook'))->toBeFalse()
            ->and($service->isValidProvider('huggingface'))->toBeFalse()
            ->and($service->isValidProvider(''))->toBeFalse();
    });

    it('rejects not_agent as a provider (it is a sentinel value, not a provider)', function () {
        $service = new AgentDetection;

        expect($service->isValidProvider('not_agent'))->toBeFalse();
    });

    it('returns all valid providers as an array', function () {
        $service = new AgentDetection;
        $providers = $service->getValidProviders();

        expect($providers)
            ->toContain('anthropic')
            ->toContain('openai')
            ->toContain('google')
            ->toContain('meta')
            ->toContain('mistral')
            ->toContain('local')
            ->toContain('unknown');
    });
});

// =========================================================================
// isAgentUserAgent Shorthand
// =========================================================================

describe('isAgentUserAgent shorthand', function () {
    it('returns true for known AI agent User-Agents', function () {
        $service = new AgentDetection;

        expect($service->isAgentUserAgent('claude-code/1.0'))->toBeTrue()
            ->and($service->isAgentUserAgent('OpenAI Python/1.0'))->toBeTrue()
            ->and($service->isAgentUserAgent('Gemini API/2.0'))->toBeTrue();
    });

    it('returns false for browser User-Agents', function () {
        $service = new AgentDetection;
        $browserUA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0';

        expect($service->isAgentUserAgent($browserUA))->toBeFalse();
    });

    it('returns false for crawler User-Agents', function () {
        $service = new AgentDetection;

        expect($service->isAgentUserAgent('Googlebot/2.1'))->toBeFalse()
            ->and($service->isAgentUserAgent('curl/7.68.0'))->toBeFalse();
    });

    it('returns true for null User-Agent (unknown programmatic access)', function () {
        $service = new AgentDetection;
        // Null UA returns unknownAgent; isAgent() is true because provider !== 'not_agent'
        expect($service->isAgentUserAgent(null))->toBeTrue();
    });

    it('returns true for unrecognised non-browser User-Agent', function () {
        $service = new AgentDetection;
        // No browser indicators → unknownAgent → isAgent() true
        expect($service->isAgentUserAgent('custom-agent/0.1'))->toBeTrue();
    });
});
