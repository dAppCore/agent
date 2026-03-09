<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Support\AgentIdentity;
use Illuminate\Http\Request;

/**
 * Service for detecting AI agents from HTTP requests.
 *
 * Identifies AI agent providers (Anthropic, OpenAI, Google, etc.) from:
 * - User-Agent string patterns
 * - MCP token headers
 * - Absence of typical browser indicators
 *
 * Part of the Trees for Agents system for rewarding AI agent referrals.
 *
 * Detection priority (highest to lowest):
 * 1. MCP token header (X-MCP-Token) — registered agents with explicit identity
 * 2. User-Agent provider patterns — matches known AI client strings
 * 3. Non-agent bot patterns — rules out search crawlers and monitoring tools
 * 4. Browser indicators — rules out real browser traffic
 * 5. Unknown agent fallback — programmatic access with no identifying UA
 *
 * Usage:
 * ```php
 * $detection = app(AgentDetection::class);
 *
 * // From a full HTTP request (checks MCP token first, then User-Agent)
 * $identity = $detection->identify($request);
 *
 * // From a User-Agent string directly
 * $identity = $detection->identifyFromUserAgent('claude-code/1.0 anthropic-api');
 *
 * // Quick boolean check
 * if ($detection->isAgent($request)) {
 *     // credit the referral tree
 * }
 *
 * // Inspect the result
 * echo $identity->provider;    // e.g. "anthropic"
 * echo $identity->model;       // e.g. "claude-sonnet" or null
 * echo $identity->confidence;  // e.g. "high"
 * echo $identity->isAgent();   // true / false
 * ```
 */
class AgentDetection
{
    /**
     * User-Agent patterns for known AI providers.
     *
     * Each entry maps a provider key to an array of detection patterns and optional
     * model-specific sub-patterns. Patterns are tested in order; the first match wins.
     *
     * Provider patterns (case-insensitive):
     *
     * - anthropic:
     *   Examples: "claude-code/1.0", "Anthropic-API/2.0 claude-sonnet",
     *             "Claude AI Assistant/1.0", "claude code (agentic)"
     *
     * - openai:
     *   Examples: "ChatGPT-User/1.0", "OpenAI/1.0 python-httpx/0.26",
     *             "GPT-4-turbo/2024-04", "o1-preview/2024-09", "o1-mini/1.0"
     *
     * - google:
     *   Examples: "Google-AI/1.0", "Gemini/1.5-pro", "Google Bard/0.1",
     *             "PaLM API/1.0 google-generativeai/0.3"
     *
     * - meta:
     *   Examples: "Meta AI/1.0", "LLaMA/2.0 meta-ai", "Llama-3/2024-04",
     *             "Llama-2-chat/70B"
     *
     * - mistral:
     *   Examples: "Mistral/0.1.0 mistralai-python/0.1", "Mixtral-8x7B/1.0",
     *             "MistralAI-Large/latest"
     *
     * Model patterns narrow the detection to a specific model variant within a provider
     * when the User-Agent includes version/model information.
     *
     * @var array<string, array{patterns: string[], model_patterns: array<string, string>}>
     */
    protected const PROVIDER_PATTERNS = [
        'anthropic' => [
            'patterns' => [
                '/claude[\s\-_]?code/i',       // e.g. "claude-code/1.0", "claude code"
                '/\banthopic\b/i',              // e.g. "Anthropic/1.0" (intentional typo tolerance)
                '/\banthropic[\s\-_]?api\b/i',  // e.g. "Anthropic-API/2.0"
                '/\bclaude\b.*\bai\b/i',        // e.g. "Claude AI Assistant/1.0"
                '/\bclaude\b.*\bassistant\b/i', // e.g. "Claude-Assistant/2.1"
            ],
            'model_patterns' => [
                'claude-opus' => '/claude[\s\-_]?opus/i',   // e.g. "claude-opus-4-5"
                'claude-sonnet' => '/claude[\s\-_]?sonnet/i', // e.g. "claude-sonnet-4-6"
                'claude-haiku' => '/claude[\s\-_]?haiku/i',  // e.g. "claude-haiku-4-5"
            ],
        ],
        'openai' => [
            'patterns' => [
                '/\bChatGPT\b/i',           // e.g. "ChatGPT-User/1.0"
                '/\bOpenAI\b/i',            // e.g. "OpenAI/1.0 python-httpx/0.26"
                '/\bGPT[\s\-_]?4\b/i',      // e.g. "GPT-4-turbo/2024-04"
                '/\bGPT[\s\-_]?3\.?5\b/i',  // e.g. "GPT-3.5-turbo/1.0"
                '/\bo1[\s\-_]?preview\b/i', // e.g. "o1-preview/2024-09"
                '/\bo1[\s\-_]?mini\b/i',    // e.g. "o1-mini/1.0"
            ],
            'model_patterns' => [
                'gpt-4' => '/\bGPT[\s\-_]?4/i',           // e.g. "GPT-4o", "GPT-4-turbo"
                'gpt-3.5' => '/\bGPT[\s\-_]?3\.?5/i',       // e.g. "GPT-3.5-turbo"
                'o1' => '/\bo1[\s\-_]?(preview|mini)?\b/i', // e.g. "o1", "o1-preview", "o1-mini"
            ],
        ],
        'google' => [
            'patterns' => [
                '/\bGoogle[\s\-_]?AI\b/i', // e.g. "Google-AI/1.0"
                '/\bGemini\b/i',           // e.g. "Gemini/1.5-pro", "gemini-flash"
                '/\bBard\b/i',             // e.g. "Google Bard/0.1" (legacy)
                '/\bPaLM\b/i',             // e.g. "PaLM API/1.0" (legacy)
            ],
            'model_patterns' => [
                'gemini-pro' => '/gemini[\s\-_]?(1\.5[\s\-_]?)?pro/i',   // e.g. "gemini-1.5-pro"
                'gemini-ultra' => '/gemini[\s\-_]?(1\.5[\s\-_]?)?ultra/i', // e.g. "gemini-ultra"
                'gemini-flash' => '/gemini[\s\-_]?(1\.5[\s\-_]?)?flash/i', // e.g. "gemini-1.5-flash"
            ],
        ],
        'meta' => [
            'patterns' => [
                '/\bMeta[\s\-_]?AI\b/i',    // e.g. "Meta AI/1.0"
                '/\bLLaMA\b/i',             // e.g. "LLaMA/2.0 meta-ai"
                '/\bLlama[\s\-_]?[23]\b/i', // e.g. "Llama-3/2024-04", "Llama-2-chat"
            ],
            'model_patterns' => [
                'llama-3' => '/llama[\s\-_]?3/i', // e.g. "Llama-3-8B", "llama3-70b"
                'llama-2' => '/llama[\s\-_]?2/i', // e.g. "Llama-2-chat/70B"
            ],
        ],
        'mistral' => [
            'patterns' => [
                '/\bMistral\b/i', // e.g. "Mistral/0.1.0 mistralai-python/0.1"
                '/\bMixtral\b/i', // e.g. "Mixtral-8x7B/1.0"
            ],
            'model_patterns' => [
                'mistral-large' => '/mistral[\s\-_]?large/i',  // e.g. "mistral-large-latest"
                'mistral-medium' => '/mistral[\s\-_]?medium/i', // e.g. "mistral-medium"
                'mixtral' => '/mixtral/i',               // e.g. "Mixtral-8x7B-Instruct"
            ],
        ],
    ];

    /**
     * Patterns that indicate a typical web browser.
     *
     * If none of these tokens appear in a User-Agent string, the request is likely
     * programmatic (a script, CLI tool, or potential agent). The patterns cover all
     * major browser families and legacy rendering engine identifiers.
     *
     * Examples of matching User-Agents:
     * - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0"
     * - "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) ... Safari/537.36"
     * - "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0"
     * - "Mozilla/5.0 ... Edg/120.0"        — Microsoft Edge (Chromium)
     * - "Opera/9.80 ... OPR/106.0"         — Opera
     * - "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1)" — Internet Explorer
     * - "Mozilla/5.0 ... Trident/7.0; rv:11.0" — IE 11 (Trident engine)
     */
    protected const BROWSER_INDICATORS = [
        '/\bMozilla\b/i', // All Gecko/WebKit/Blink browsers include "Mozilla/5.0"
        '/\bChrome\b/i',  // Chrome, Chromium, and most Chromium-based browsers
        '/\bSafari\b/i',  // Safari and WebKit-based browsers
        '/\bFirefox\b/i', // Mozilla Firefox
        '/\bEdge\b/i',    // Microsoft Edge (legacy "Edge/" and Chromium "Edg/")
        '/\bOpera\b/i',   // Opera ("Opera/" classic, "OPR/" modern)
        '/\bMSIE\b/i',    // Internet Explorer (e.g. "MSIE 11.0")
        '/\bTrident\b/i', // IE 11 Trident rendering engine token
    ];

    /**
     * Known bot patterns that are NOT AI agents.
     *
     * These should resolve to `AgentIdentity::notAnAgent()` rather than
     * `AgentIdentity::unknownAgent()`, because we can positively identify them
     * as a specific non-AI automated client (crawler, monitoring, HTTP library, etc.).
     *
     * Categories and example User-Agents:
     *
     * Search engine crawlers:
     * - "Googlebot/2.1 (+http://www.google.com/bot.html)"
     * - "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)"
     * - "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)"
     * - "DuckDuckBot/1.0; (+http://duckduckgo.com/duckduckbot.html)"
     * - "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)"
     * - "Applebot/0.1 (+http://www.apple.com/go/applebot)"
     *
     * Social media / link-preview bots:
     * - "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
     * - "Twitterbot/1.0"
     * - "LinkedInBot/1.0 (compatible; Mozilla/5.0; Apache-HttpClient/4.5)"
     * - "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)"
     * - "DiscordBot (https://discordapp.com) 1.0"
     * - "TelegramBot (like TwitterBot)"
     * - "WhatsApp/2.23.20 A"
     *
     * SEO / analytics crawlers:
     * - "Mozilla/5.0 (compatible; SemrushBot/7~bl; +http://www.semrush.com/bot.html)"
     * - "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)"
     *
     * Generic HTTP clients (scripts, developer tools):
     * - "curl/7.88.1"
     * - "Wget/1.21.4"
     * - "python-requests/2.31.0"
     * - "Go-http-client/2.0"
     * - "PostmanRuntime/7.35.0"
     * - "insomnia/2023.5.8"
     * - "axios/1.6.0"
     * - "node-fetch/2.6.11"
     *
     * Uptime / monitoring services:
     * - "UptimeRobot/2.0 (+http://www.uptimerobot.com/)"
     * - "Pingdom.com_bot_version_1.4 (http://www.pingdom.com/)"
     * - "Datadog Agent/7.45.0"
     * - "NewRelicPinger/v1 AccountId=12345"
     */
    protected const NON_AGENT_BOTS = [
        // Search engine crawlers
        '/\bGooglebot\b/i',
        '/\bBingbot\b/i',
        '/\bYandexBot\b/i',
        '/\bDuckDuckBot\b/i',
        '/\bBaiduspider\b/i',
        '/\bApplebot\b/i',

        // Social media / link-preview bots
        '/\bfacebookexternalhit\b/i',
        '/\bTwitterbot\b/i',
        '/\bLinkedInBot\b/i',
        '/\bSlackbot\b/i',
        '/\bDiscordBot\b/i',
        '/\bTelegramBot\b/i',
        '/\bWhatsApp\//i',

        // SEO / analytics crawlers
        '/\bSEMrushBot\b/i',
        '/\bAhrefsBot\b/i',

        // Generic HTTP clients
        '/\bcurl\b/i',
        '/\bwget\b/i',
        '/\bpython-requests\b/i',
        '/\bgo-http-client\b/i',
        '/\bPostman/i',
        '/\bInsomnia\b/i',
        '/\baxios\b/i',
        '/\bnode-fetch\b/i',

        // Uptime / monitoring services
        '/\bUptimeRobot\b/i',
        '/\bPingdom\b/i',
        '/\bDatadog\b/i',
        '/\bNewRelic\b/i',
    ];

    /**
     * The MCP token header used to identify registered AI agents.
     *
     * Agents send this header to bypass User-Agent heuristics and declare their
     * identity explicitly. Two token formats are supported:
     *
     * - Opaque AgentApiKey token (prefix "ak_"):
     *   Looked up in the database. Grants highest confidence when the key is active.
     *   Example: `X-MCP-Token: ak_a1b2c3d4e5f6...`
     *
     * - Structured provider:model:secret token:
     *   Encodes provider and model directly in the token value.
     *   Example: `X-MCP-Token: anthropic:claude-sonnet:mysecret`
     *   Example: `X-MCP-Token: openai:gpt-4:xyz789`
     */
    protected const MCP_TOKEN_HEADER = 'X-MCP-Token';

    /**
     * Identify an agent from an HTTP request.
     */
    public function identify(Request $request): AgentIdentity
    {
        // First, check for MCP token (highest priority)
        $mcpToken = $request->header(self::MCP_TOKEN_HEADER);
        if ($mcpToken) {
            return $this->identifyFromMcpToken($mcpToken);
        }

        // Then check User-Agent
        $userAgent = $request->userAgent();

        return $this->identifyFromUserAgent($userAgent);
    }

    /**
     * Identify an agent from a User-Agent string.
     */
    public function identifyFromUserAgent(?string $userAgent): AgentIdentity
    {
        if (! $userAgent || trim($userAgent) === '') {
            // Empty User-Agent is suspicious but not definitive
            return AgentIdentity::unknownAgent();
        }

        // Check for known AI providers first (highest confidence)
        foreach (self::PROVIDER_PATTERNS as $provider => $config) {
            foreach ($config['patterns'] as $pattern) {
                if (preg_match($pattern, $userAgent)) {
                    $model = $this->detectModel($userAgent, $config['model_patterns']);

                    return $this->createProviderIdentity($provider, $model, AgentIdentity::CONFIDENCE_HIGH);
                }
            }
        }

        // Check for non-agent bots (search engines, monitoring, etc.)
        foreach (self::NON_AGENT_BOTS as $pattern) {
            if (preg_match($pattern, $userAgent)) {
                return AgentIdentity::notAnAgent();
            }
        }

        // Check if it looks like a normal browser
        if ($this->looksLikeBrowser($userAgent)) {
            return AgentIdentity::notAnAgent();
        }

        // No browser indicators and not a known bot — might be an unknown agent
        return AgentIdentity::unknownAgent();
    }

    /**
     * Identify an agent from an MCP token.
     *
     * MCP tokens can encode provider and model information for registered agents.
     * Supports two token formats:
     * - Structured: "provider:model:secret" (e.g., "anthropic:claude-opus:abc123")
     * - Opaque: "ak_xxxx..." (registered AgentApiKey, looked up in database)
     */
    public function identifyFromMcpToken(string $token): AgentIdentity
    {
        // Check for opaque token format (AgentApiKey)
        // AgentApiKey tokens start with "ak_" prefix
        if (str_starts_with($token, 'ak_')) {
            return $this->identifyFromAgentApiKey($token);
        }

        // Try structured token format: "provider:model:secret"
        // Expected token formats:
        // - "anthropic:claude-opus:abc123" (provider:model:secret)
        // - "openai:gpt-4:xyz789"
        $parts = explode(':', $token, 3);

        if (count($parts) >= 2) {
            $provider = strtolower($parts[0]);
            $model = $parts[1] ?? null;

            // Validate provider is in our known list
            if ($this->isValidProvider($provider)) {
                return $this->createProviderIdentity($provider, $model, AgentIdentity::CONFIDENCE_HIGH);
            }
        }

        // Unrecognised token format — return unknown with medium confidence
        // (token present suggests agent, but we cannot identify provider)
        return new AgentIdentity('unknown', null, AgentIdentity::CONFIDENCE_MEDIUM);
    }

    /**
     * Identify an agent from a registered AgentApiKey token.
     *
     * Looks up the token in the database and extracts provider/model
     * from the key's metadata if available.
     */
    protected function identifyFromAgentApiKey(string $token): AgentIdentity
    {
        $apiKey = AgentApiKey::findByKey($token);

        if ($apiKey === null) {
            // Token not found in database — invalid or revoked
            return AgentIdentity::unknownAgent();
        }

        // Check if the key is active
        if (! $apiKey->isActive()) {
            // Expired or revoked key — still an agent, but unknown
            return AgentIdentity::unknownAgent();
        }

        // Extract provider and model from key name or permissions
        // Key names often follow pattern: "Claude Opus Agent" or "GPT-4 Integration"
        $provider = $this->extractProviderFromKeyName($apiKey->name);
        $model = $this->extractModelFromKeyName($apiKey->name);

        if ($provider !== null) {
            return $this->createProviderIdentity($provider, $model, AgentIdentity::CONFIDENCE_HIGH);
        }

        // Valid key but cannot determine provider — return unknown with high confidence
        // (we know it's a registered agent, just not which provider)
        return new AgentIdentity('unknown', null, AgentIdentity::CONFIDENCE_HIGH);
    }

    /**
     * Extract provider from an API key name.
     *
     * Attempts to identify provider from common naming patterns:
     * - "Claude Agent", "Anthropic Integration" => anthropic
     * - "GPT-4 Agent", "OpenAI Integration" => openai
     * - "Gemini Agent", "Google AI" => google
     */
    protected function extractProviderFromKeyName(string $name): ?string
    {
        $nameLower = strtolower($name);

        // Check for provider keywords
        $providerPatterns = [
            'anthropic' => ['anthropic', 'claude'],
            'openai' => ['openai', 'gpt', 'chatgpt', 'o1-'],
            'google' => ['google', 'gemini', 'bard', 'palm'],
            'meta' => ['meta', 'llama'],
            'mistral' => ['mistral', 'mixtral'],
        ];

        foreach ($providerPatterns as $provider => $keywords) {
            foreach ($keywords as $keyword) {
                if (str_contains($nameLower, $keyword)) {
                    return $provider;
                }
            }
        }

        return null;
    }

    /**
     * Extract model from an API key name.
     *
     * Attempts to identify specific model from naming patterns:
     * - "Claude Opus Agent" => claude-opus
     * - "GPT-4 Integration" => gpt-4
     */
    protected function extractModelFromKeyName(string $name): ?string
    {
        $nameLower = strtolower($name);

        // Check for model keywords
        $modelPatterns = [
            'claude-opus' => ['opus'],
            'claude-sonnet' => ['sonnet'],
            'claude-haiku' => ['haiku'],
            'gpt-4' => ['gpt-4', 'gpt4'],
            'gpt-3.5' => ['gpt-3.5', 'gpt3.5', 'turbo'],
            'o1' => ['o1-preview', 'o1-mini', 'o1 '],
            'gemini-pro' => ['gemini pro', 'gemini-pro'],
            'gemini-flash' => ['gemini flash', 'gemini-flash'],
            'llama-3' => ['llama 3', 'llama-3', 'llama3'],
        ];

        foreach ($modelPatterns as $model => $keywords) {
            foreach ($keywords as $keyword) {
                if (str_contains($nameLower, $keyword)) {
                    return $model;
                }
            }
        }

        return null;
    }

    /**
     * Check if the User-Agent looks like a normal web browser.
     */
    protected function looksLikeBrowser(?string $userAgent): bool
    {
        if (! $userAgent) {
            return false;
        }

        foreach (self::BROWSER_INDICATORS as $pattern) {
            if (preg_match($pattern, $userAgent)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Detect the model from User-Agent patterns.
     *
     * @param  array<string, string>  $modelPatterns
     */
    protected function detectModel(string $userAgent, array $modelPatterns): ?string
    {
        foreach ($modelPatterns as $model => $pattern) {
            if (preg_match($pattern, $userAgent)) {
                return $model;
            }
        }

        return null;
    }

    /**
     * Create an identity for a known provider.
     */
    protected function createProviderIdentity(string $provider, ?string $model, string $confidence): AgentIdentity
    {
        return match ($provider) {
            'anthropic' => AgentIdentity::anthropic($model, $confidence),
            'openai' => AgentIdentity::openai($model, $confidence),
            'google' => AgentIdentity::google($model, $confidence),
            'meta' => AgentIdentity::meta($model, $confidence),
            'mistral' => AgentIdentity::mistral($model, $confidence),
            'local' => AgentIdentity::local($model, $confidence),
            default => new AgentIdentity($provider, $model, $confidence),
        };
    }

    /**
     * Check if a provider name is valid.
     */
    public function isValidProvider(string $provider): bool
    {
        return in_array($provider, [
            'anthropic',
            'openai',
            'google',
            'meta',
            'mistral',
            'local',
            'unknown',
        ], true);
    }

    /**
     * Get the list of valid providers.
     *
     * @return string[]
     */
    public function getValidProviders(): array
    {
        return [
            'anthropic',
            'openai',
            'google',
            'meta',
            'mistral',
            'local',
            'unknown',
        ];
    }

    /**
     * Check if a request appears to be from an AI agent.
     */
    public function isAgent(Request $request): bool
    {
        return $this->identify($request)->isAgent();
    }

    /**
     * Check if a User-Agent appears to be from an AI agent.
     */
    public function isAgentUserAgent(?string $userAgent): bool
    {
        return $this->identifyFromUserAgent($userAgent)->isAgent();
    }
}
