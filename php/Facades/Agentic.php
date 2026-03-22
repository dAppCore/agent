<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Facades;

use Core\Mod\Agentic\Services\AgenticManager;
use Illuminate\Support\Facades\Facade;

/**
 * @method static \Core\Mod\Agentic\Services\AgenticProviderInterface provider(string $name = null)
 * @method static \Core\Mod\Agentic\Services\ClaudeService claude()
 * @method static \Core\Mod\Agentic\Services\GeminiService gemini()
 * @method static \Core\Mod\Agentic\Services\OpenAIService openai()
 * @method static array availableProviders()
 * @method static bool isAvailable(string $name)
 * @method static void setDefault(string $name)
 *
 * @see \Core\Mod\Agentic\Services\AgenticManager
 */
class Agentic extends Facade
{
    protected static function getFacadeAccessor(): string
    {
        return AgenticManager::class;
    }
}
