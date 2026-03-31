<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Subscription;

use Core\Actions\Action;

class DetectCapabilities
{
    use Action;

    /**
     * @param  array<string, string>  $apiKeys
     * @return array<string, mixed>
     */
    public function handle(array $apiKeys = []): array
    {
        $resolved = [
            'claude' => ($apiKeys['claude'] ?? '') !== '' || (string) config('agentic.claude.api_key', '') !== '',
            'gemini' => ($apiKeys['gemini'] ?? '') !== '' || (string) config('agentic.gemini.api_key', '') !== '',
            'openai' => ($apiKeys['openai'] ?? '') !== '' || (string) config('agentic.openai.api_key', '') !== '',
        ];

        return [
            'providers' => $resolved,
            'available' => array_keys(array_filter($resolved)),
        ];
    }
}
