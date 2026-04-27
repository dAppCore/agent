<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;

class HermesClient
{
    /**
     * Create a Hermes response run for a Mantis ticket prompt.
     *
     * @param  array<string, mixed>  $metadata
     * @return array<string, mixed>
     */
    public function createResponse(AgentProfile $profile, string $input, array $metadata = []): array
    {
        $response = $this->request($profile)->post($this->url($profile), [
            'model' => 'default',
            'input' => $input,
            'metadata' => $metadata,
        ]);

        $response->throw();

        $payload = $response->json();

        return is_array($payload) ? $payload : [];
    }

    private function request(AgentProfile $profile): PendingRequest
    {
        return Http::withToken((string) $profile->api_key_cipher)
            ->acceptJson()
            ->asJson()
            ->timeout(60);
    }

    private function url(AgentProfile $profile): string
    {
        return rtrim($profile->gateway_url, '/').'/v1/responses';
    }
}
