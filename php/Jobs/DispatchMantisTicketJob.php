<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Jobs;

use Core\Mod\Agentic\Models\AgentDispatch;
use Core\Mod\Agentic\Models\AgentProfile;
use Core\Mod\Agentic\Services\HermesClient;
use Core\Mod\Agentic\Services\MantisClient;
use Core\Mod\Agentic\Services\ProfileSelector;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Log;
use RuntimeException;

class DispatchMantisTicketJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $tries = 5;

    public int $backoff = 60;

    public int $timeout = 120;

    public function __construct(
        public int $ticketId,
    ) {
        $this->onQueue('ai');
    }

    public function handle(
        MantisClient $mantisClient,
        HermesClient $hermesClient,
        ?ProfileSelector $profileSelector = null,
    ): void {
        $profileSelector ??= $this->resolveProfileSelector();
        $ticket = $mantisClient->get($this->ticketId);

        $profile = $this->pickProfile($profileSelector, $ticket);

        if ($profile === null) {
            Log::warning('DispatchMantisTicketJob: no agent profile available for Mantis ticket', [
                'ticket_id' => $this->ticketId,
            ]);

            $this->release(60);

            return;
        }

        $response = $hermesClient->createResponse(
            $profile,
            $this->buildPrompt($ticket),
            [
                'conversation' => "mantis-{$this->ticketId}",
                'ticket_id' => $this->ticketId,
            ],
        );

        $dispatch = $this->createDispatch($profile, $response);

        CaptureDispatchResultJob::dispatch($this->ticketId, $dispatch->response_id);
    }

    /**
     * @return array<int, string>
     */
    public function tags(): array
    {
        return [
            'mantis-dispatch',
            "ticket:{$this->ticketId}",
        ];
    }

    private function resolveProfileSelector(): ?ProfileSelector
    {
        if (! class_exists(ProfileSelector::class)) {
            return null;
        }

        return app(ProfileSelector::class);
    }

    private function pickProfile(?ProfileSelector $profileSelector, mixed $ticket): ?AgentProfile
    {
        if ($profileSelector === null) {
            return null;
        }

        $profile = $profileSelector->pickFor($ticket);

        return $profile instanceof AgentProfile ? $profile : null;
    }

    private function createDispatch(AgentProfile $profile, array $response): AgentDispatch
    {
        return AgentDispatch::query()->create([
            'ticket_id' => $this->ticketId,
            'profile_id' => $profile->getKey() === null ? null : (int) $profile->getKey(),
            'response_id' => $this->extractResponseId($response),
            'run_id' => $this->extractRunId($response),
            'status' => $this->extractStatus($response),
        ]);
    }

    private function buildPrompt(mixed $ticket): string
    {
        foreach (['body', 'description', 'text', 'content', 'summary', 'title'] as $key) {
            $value = data_get($ticket, $key);

            if (is_string($value) && trim($value) !== '') {
                return trim($value);
            }
        }

        if (is_string($ticket) && trim($ticket) !== '') {
            return trim($ticket);
        }

        $fallback = json_encode($ticket, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);

        if (is_string($fallback) && $fallback !== '') {
            return $fallback;
        }

        return "Mantis ticket {$this->ticketId}";
    }

    private function extractResponseId(array $response): string
    {
        $responseId = data_get($response, 'id');

        if (is_scalar($responseId) && $responseId !== '') {
            return (string) $responseId;
        }

        throw new RuntimeException('Hermes response did not include an id');
    }

    private function extractRunId(array $response): ?string
    {
        $runId = data_get($response, 'run_id') ?? data_get($response, 'run.id');

        return is_scalar($runId) && $runId !== '' ? (string) $runId : null;
    }

    private function extractStatus(array $response): string
    {
        $status = data_get($response, 'status');

        return is_string($status) && $status !== ''
            ? $status
            : AgentDispatch::STATUS_QUEUED;
    }
}
