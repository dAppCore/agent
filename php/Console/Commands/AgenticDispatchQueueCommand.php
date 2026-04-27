<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Jobs\DispatchMantisTicketJob;
use Core\Mod\Agentic\Services\MantisClient;
use Core\Mod\Agentic\Services\ProfileSelector;
use Illuminate\Console\Command;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Log;
use Throwable;

class AgenticDispatchQueueCommand extends Command
{
    private const IN_FLIGHT_MARKER_PREFIX = 'Picked up by agentic dispatch queue';

    private const IN_FLIGHT_WINDOW_MINUTES = 5;

    protected $signature = 'agentic:dispatch-queue
        {--limit=3 : Maximum Mantis tickets to queue}';

    protected $description = 'Queue open Mantis tickets for agentic dispatch';

    public function handle(MantisClient $mantisClient, ProfileSelector $profileSelector): int
    {
        $limit = (int) $this->option('limit');

        if ($limit < 1) {
            $this->error('--limit must be greater than zero.');

            return self::FAILURE;
        }

        $tickets = $mantisClient->listOpen();

        if ($tickets === []) {
            $this->info('No open Mantis tickets awaiting dispatch.');
            Log::info('agentic:dispatch-queue found no open tickets', [
                'limit' => $limit,
            ]);

            return self::SUCCESS;
        }

        Log::info('agentic:dispatch-queue starting', [
            'limit' => $limit,
            'open_tickets' => count($tickets),
        ]);

        $queued = 0;
        $skippedAssigned = 0;
        $skippedSuppressed = 0;
        $skippedUnavailable = 0;
        $failed = 0;

        foreach ($tickets as $ticket) {
            if ($queued >= $limit) {
                break;
            }

            $ticketId = $this->ticketId($ticket);

            if ($ticketId === null) {
                $failed++;

                Log::warning('agentic:dispatch-queue skipped ticket with invalid identifier', [
                    'ticket' => $ticket,
                ]);

                continue;
            }

            if (! $this->isUnassigned($ticket)) {
                $skippedAssigned++;

                Log::info('agentic:dispatch-queue skipped assigned ticket', [
                    'ticket_id' => $ticketId,
                ]);

                continue;
            }

            if ($this->hasRecentDispatchMarker($ticket)) {
                $skippedSuppressed++;

                Log::info('agentic:dispatch-queue skipped in-flight ticket', [
                    'ticket_id' => $ticketId,
                ]);

                continue;
            }

            $profile = $profileSelector->pickFor($ticket);

            if ($profile === null) {
                $skippedUnavailable++;

                Log::info('agentic:dispatch-queue found no eligible profile for ticket', [
                    'ticket_id' => $ticketId,
                ]);

                continue;
            }

            try {
                $mantisClient->note($ticketId, $this->inFlightNote($profile->name));
                DispatchMantisTicketJob::dispatch($ticketId);

                $queued++;

                $this->line("Queued Mantis ticket #{$ticketId} with profile {$profile->name}.");
                Log::info('agentic:dispatch-queue queued ticket', [
                    'ticket_id' => $ticketId,
                    'profile_id' => $profile->id,
                    'profile_name' => $profile->name,
                ]);
            } catch (Throwable $throwable) {
                $failed++;

                $this->warn("Failed to queue Mantis ticket #{$ticketId}: {$throwable->getMessage()}");
                Log::warning('agentic:dispatch-queue failed to queue ticket', [
                    'ticket_id' => $ticketId,
                    'profile_id' => $profile->id,
                    'profile_name' => $profile->name,
                    'error' => $throwable->getMessage(),
                ]);
            }
        }

        $this->info(
            "Queued {$queued} Mantis ticket(s); "
            ."skipped {$skippedAssigned} assigned, {$skippedSuppressed} in-flight, "
            ."{$skippedUnavailable} without matching profile; {$failed} failed.",
        );

        Log::info('agentic:dispatch-queue completed', [
            'queued' => $queued,
            'skipped_assigned' => $skippedAssigned,
            'skipped_suppressed' => $skippedSuppressed,
            'skipped_unavailable' => $skippedUnavailable,
            'failed' => $failed,
        ]);

        return $failed === 0 ? self::SUCCESS : self::FAILURE;
    }

    /**
     * @param  array<string, mixed>  $ticket
     */
    private function ticketId(array $ticket): ?int
    {
        $ticketId = data_get($ticket, 'id');

        if (is_int($ticketId)) {
            return $ticketId;
        }

        if (is_string($ticketId) && ctype_digit($ticketId)) {
            return (int) $ticketId;
        }

        return null;
    }

    /**
     * @param  array<string, mixed>  $ticket
     */
    private function isUnassigned(array $ticket): bool
    {
        foreach ([
            data_get($ticket, 'handler.id'),
            data_get($ticket, 'handler.name'),
            data_get($ticket, 'handler.username'),
            data_get($ticket, 'assigned_to.id'),
            data_get($ticket, 'assigned_to.name'),
            data_get($ticket, 'assigned_to.username'),
        ] as $candidate) {
            if (is_scalar($candidate) && trim((string) $candidate) !== '') {
                return false;
            }
        }

        return blank(data_get($ticket, 'handler')) && blank(data_get($ticket, 'assigned_to'));
    }

    /**
     * @param  array<string, mixed>  $ticket
     */
    private function hasRecentDispatchMarker(array $ticket): bool
    {
        $notes = data_get($ticket, 'notes');

        if (! is_array($notes)) {
            return false;
        }

        $threshold = now()->subMinutes(self::IN_FLIGHT_WINDOW_MINUTES);

        foreach ($notes as $note) {
            $noteText = $this->noteText($note);
            $createdAt = $this->noteTimestamp($note);

            if ($noteText === null || $createdAt === null) {
                continue;
            }

            if (str_contains($noteText, self::IN_FLIGHT_MARKER_PREFIX) && $createdAt->greaterThanOrEqualTo($threshold)) {
                return true;
            }
        }

        return false;
    }

    private function inFlightNote(string $profileName): string
    {
        return self::IN_FLIGHT_MARKER_PREFIX
            ." (profile: {$profileName}). Suppressing concurrent dispatch for 5min.";
    }

    private function noteText(mixed $note): ?string
    {
        $text = data_get($note, 'text');

        if (! is_string($text)) {
            return null;
        }

        $text = trim($text);

        return $text === '' ? null : $text;
    }

    private function noteTimestamp(mixed $note): ?Carbon
    {
        foreach (['created_at', 'date_submitted', 'updated_at', 'last_modified'] as $key) {
            $timestamp = $this->parseTimestamp(data_get($note, $key));

            if ($timestamp instanceof Carbon) {
                return $timestamp;
            }
        }

        return null;
    }

    private function parseTimestamp(mixed $value): ?Carbon
    {
        if (is_int($value) || (is_string($value) && ctype_digit($value))) {
            return Carbon::createFromTimestamp((int) $value);
        }

        if (is_string($value) && trim($value) !== '') {
            try {
                return Carbon::parse($value);
            } catch (Throwable) {
                return null;
            }
        }

        foreach (['date', 'value', 'timestamp'] as $key) {
            $nestedValue = data_get($value, $key);

            if ($nestedValue === null) {
                continue;
            }

            $timestamp = $this->parseTimestamp($nestedValue);

            if ($timestamp instanceof Carbon) {
                return $timestamp;
            }
        }

        return null;
    }
}
