<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;
use RuntimeException;

class MantisClient
{
    public function __construct(
        private ?string $baseUrl = null,
        private ?string $token = null,
    ) {}

    public function note(int $ticketId, string $text): void
    {
        $response = $this->request()->post("/api/rest/issues/{$ticketId}/notes", [
            'text' => $text,
        ]);

        if (! $response->successful()) {
            throw new RuntimeException("Mantis note failed: {$response->status()}");
        }
    }

    public function close(int $ticketId, string $resolution = 'fixed'): void
    {
        $response = $this->request()->patch("/api/rest/issues/{$ticketId}", [
            'status' => [
                'name' => 'closed',
            ],
            'resolution' => [
                'name' => $resolution,
            ],
        ]);

        if (! $response->successful()) {
            throw new RuntimeException("Mantis close failed: {$response->status()}");
        }
    }

    private function request(): PendingRequest
    {
        return Http::acceptJson()
            ->baseUrl($this->resolveBaseUrl())
            ->withHeaders([
                'Authorization' => $this->resolveToken(),
            ])
            ->timeout(15);
    }

    private function resolveBaseUrl(): string
    {
        return rtrim(
            $this->baseUrl ?? (string) config('agentic.mantis.base_url', 'https://tasks.lthn.sh'),
            '/',
        );
    }

    private function resolveToken(): string
    {
        return $this->token ?? (string) config('agentic.mantis.token');
    }
}
