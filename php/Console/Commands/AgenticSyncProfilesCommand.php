<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use RuntimeException;
use Throwable;

class AgenticSyncProfilesCommand extends Command
{
    protected $signature = 'agentic:sync-profiles';

    protected $description = 'Synchronise agent profile capability tags from gateway model catalogues';

    public function handle(): int
    {
        $profiles = AgentProfile::query()
            ->orderBy('id')
            ->get();

        if ($profiles->isEmpty()) {
            $this->info('No agent profiles found to synchronise.');
            Log::info('agentic:sync-profiles found no profiles');

            return self::SUCCESS;
        }

        Log::info('agentic:sync-profiles starting', [
            'profiles' => $profiles->count(),
        ]);

        $synchronised = 0;
        $failed = 0;

        foreach ($profiles as $profile) {
            try {
                $modelIdentifiers = $this->fetchModelIdentifiers($profile);
                $capabilityTags = $this->inferCapabilityTags($modelIdentifiers);

                $profile->forceFill([
                    'capability_tags' => $capabilityTags,
                ])->save();

                $synchronised++;

                $this->line(
                    sprintf(
                        'Synchronised %s: %s',
                        $profile->name,
                        $capabilityTags === [] ? '[none]' : '['.implode(', ', $capabilityTags).']',
                    ),
                );

                Log::info('agentic:sync-profiles synchronised profile', [
                    'profile_id' => $profile->id,
                    'profile_name' => $profile->name,
                    'model_count' => count($modelIdentifiers),
                    'capability_tags' => $capabilityTags,
                ]);
            } catch (Throwable $throwable) {
                $failed++;

                $message = "Failed to synchronise {$profile->name}: {$throwable->getMessage()}";

                $this->warn($message);
                Log::warning('agentic:sync-profiles failed to synchronise profile', [
                    'profile_id' => $profile->id,
                    'profile_name' => $profile->name,
                    'error' => $throwable->getMessage(),
                ]);
            }
        }

        $this->info("Synchronised {$synchronised} profile(s); {$failed} failed.");

        Log::info('agentic:sync-profiles completed', [
            'synchronised' => $synchronised,
            'failed' => $failed,
        ]);

        return $failed === 0 ? self::SUCCESS : self::FAILURE;
    }

    /**
     * @return list<string>
     */
    private function fetchModelIdentifiers(AgentProfile $profile): array
    {
        $response = Http::withToken((string) $profile->api_key_cipher)
            ->acceptJson()
            ->timeout(15)
            ->get($this->modelsUrl($profile));

        if (! $response->successful()) {
            throw new RuntimeException("Gateway models request failed: {$response->status()}");
        }

        $payload = $response->json();
        $records = is_array($payload)
            ? ($payload['data'] ?? $payload['models'] ?? $payload)
            : [];

        if (! is_array($records)) {
            return [];
        }

        $identifiers = [];

        foreach ($records as $record) {
            $identifier = $this->extractModelIdentifier($record);

            if ($identifier === null) {
                continue;
            }

            $identifiers[] = $identifier;
        }

        sort($identifiers, SORT_NATURAL | SORT_FLAG_CASE);

        return array_values(array_unique($identifiers));
    }

    /**
     * @return list<string>
     */
    private function inferCapabilityTags(array $modelIdentifiers): array
    {
        $capabilityTags = [];

        foreach ($modelIdentifiers as $modelIdentifier) {
            foreach ($this->inferTagsForModel($modelIdentifier) as $tag) {
                $capabilityTags[$tag] = $tag;
            }
        }

        ksort($capabilityTags);

        return array_values($capabilityTags);
    }

    /**
     * @return list<string>
     */
    private function inferTagsForModel(string $modelIdentifier): array
    {
        $tags = [];
        $isSmallReasoner = (
            (str_contains($modelIdentifier, 'gpt') || str_contains($modelIdentifier, 'o3') || str_contains($modelIdentifier, 'o4'))
            && (str_contains($modelIdentifier, 'mini') || str_contains($modelIdentifier, 'nano'))
        );

        if (str_contains($modelIdentifier, 'embedding')) {
            $tags[] = 'embedding';
        }

        if (str_contains($modelIdentifier, 'claude-opus') || str_contains($modelIdentifier, 'opus')) {
            array_push($tags, 'analysis', 'core', 'handoff');
        } elseif (! $isSmallReasoner && (
            str_contains($modelIdentifier, 'claude')
            || str_contains($modelIdentifier, 'sonnet')
            || str_contains($modelIdentifier, 'haiku')
            || str_contains($modelIdentifier, 'gpt')
            || str_contains($modelIdentifier, 'o3')
            || str_contains($modelIdentifier, 'o4')
        )) {
            $tags[] = 'analysis';
        }

        if ($isSmallReasoner) {
            array_push($tags, 'cheap', 'dispatch');
        }

        return array_values(array_unique($tags));
    }

    private function modelsUrl(AgentProfile $profile): string
    {
        return rtrim($profile->gateway_url, '/').'/v1/models';
    }

    private function extractModelIdentifier(mixed $record): ?string
    {
        if (is_string($record)) {
            return $this->normaliseModelIdentifier($record);
        }

        foreach (['id', 'name', 'model'] as $key) {
            $value = data_get($record, $key);

            if (! is_string($value)) {
                continue;
            }

            return $this->normaliseModelIdentifier($value);
        }

        return null;
    }

    private function normaliseModelIdentifier(string $modelIdentifier): ?string
    {
        $normalised = strtolower(trim($modelIdentifier));

        return $normalised === '' ? null : $normalised;
    }
}
