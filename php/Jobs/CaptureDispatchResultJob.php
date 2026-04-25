<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Jobs;

use Core\Mod\Agentic\Services\MantisClient;
use Core\Mod\Agentic\Services\ShaExtractor;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use RuntimeException;

class CaptureDispatchResultJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    private const SUMMARY_PATHS = [
        'output.summary',
        'data.output.summary',
        'result.output.summary',
        'summary',
    ];

    private const OUTPUT_PATHS = [
        'output.content',
        'output.text',
        'data.output.content',
        'data.output.text',
        'result.output.content',
        'result.output.text',
        'content',
        'text',
        'message',
        'output',
        'data.output',
        'result.output',
    ];

    private const EVENT_PATHS = [
        'events',
        'data.events',
        'result.events',
        'log',
        'data.log',
        'result.log',
        'tail',
        'data.tail',
        'result.tail',
        'last_event',
        'data.last_event',
        'result.last_event',
    ];

    public int $tries = 3;

    public int $backoff = 60;

    public int $timeout = 120;

    /**
     * @param  array<string, mixed>  $response
     */
    public function __construct(
        public int $ticketId,
        public array $response,
        public ?string $repo = null,
        public ?\Mod\AgentProfile $profile = null,
    ) {
        $this->onQueue('ai');
    }

    public function handle(MantisClient $mantis, ShaExtractor $extractor): void
    {
        $transcript = $this->transcriptText();

        if ($transcript === '') {
            throw new RuntimeException('Hermes response did not contain model output.');
        }

        $match = $extractor->extract($transcript);
        $sha = $match['sha'];

        if ($sha === null) {
            throw new RuntimeException('Unable to extract commit SHA from model output.');
        }

        $repo = $match['repo'] ?? $this->normaliseRepo($this->repo);

        if ($repo === null) {
            throw new RuntimeException('Unable to determine repository for close note.');
        }

        $forgeUrl = $match['forge_url'] ?? $this->buildForgeUrl($repo, $sha);

        $mantis->note($this->ticketId, $this->buildCloseNote(
            sha: $sha,
            repo: $repo,
            forgeUrl: $forgeUrl,
            summaryLine: $this->firstLineOfSummary(),
            profile: $this->profile,
        ));
        $mantis->close($this->ticketId);
    }

    private function buildCloseNote(
        string $sha,
        string $repo,
        string $forgeUrl,
        string $summaryLine,
        ?\Mod\AgentProfile $profile = null,
    ): string {
        $profileName = $this->profileName($profile);

        return "Implemented by codex exec supervised by CorePHP Agentic dispatch (profile: {$profileName}).\n"
            ."Commit: {$sha} on {$repo} dev ({$forgeUrl}).\n"
            ."{$summaryLine}\n\n"
            .'Filed-by: agentic';
    }

    private function firstLineOfSummary(): string
    {
        $summary = $this->firstText(self::SUMMARY_PATHS);

        if ($summary === '') {
            $summary = $this->firstText(self::OUTPUT_PATHS);
        }

        if ($summary === '') {
            $summary = $this->transcriptText();
        }

        foreach (preg_split('/\R/', $summary) ?: [] as $line) {
            $trimmed = trim($line);

            if ($trimmed !== '') {
                return $trimmed;
            }
        }

        return 'No summary available.';
    }

    private function transcriptText(): string
    {
        $segments = [];

        foreach (array_merge(self::SUMMARY_PATHS, self::OUTPUT_PATHS, self::EVENT_PATHS) as $path) {
            $text = $this->stringifyText(data_get($this->response, $path));

            if ($text !== '') {
                $segments[] = $text;
            }
        }

        if ($segments === []) {
            $fallback = $this->stringifyText($this->response);

            if ($fallback !== '') {
                $segments[] = $fallback;
            }
        }

        return implode("\n", array_values(array_unique($segments)));
    }

    /**
     * @param  array<int, string>  $paths
     */
    private function firstText(array $paths): string
    {
        foreach ($paths as $path) {
            $text = $this->stringifyText(data_get($this->response, $path));

            if ($text !== '') {
                return $text;
            }
        }

        return '';
    }

    private function stringifyText(mixed $value): string
    {
        $segments = $this->collectStrings($value);
        $segments = array_values(array_filter(array_map(
            static fn (string $segment): string => trim($segment),
            $segments,
        ), static fn (string $segment): bool => $segment !== ''));

        return implode("\n", array_values(array_unique($segments)));
    }

    /**
     * @return array<int, string>
     */
    private function collectStrings(mixed $value): array
    {
        if (is_string($value)) {
            return [$value];
        }

        if (is_int($value) || is_float($value)) {
            return [(string) $value];
        }

        if (! is_array($value)) {
            return [];
        }

        $preferredKeys = ['summary', 'content', 'text', 'message', 'body', 'log', 'tail'];
        $segments = [];

        foreach ($preferredKeys as $key) {
            if (array_key_exists($key, $value)) {
                $segments = array_merge($segments, $this->collectStrings($value[$key]));
            }
        }

        foreach ($value as $key => $item) {
            if (in_array($key, $preferredKeys, true)) {
                continue;
            }

            $segments = array_merge($segments, $this->collectStrings($item));
        }

        return $segments;
    }

    private function normaliseRepo(?string $repo): ?string
    {
        if ($repo === null) {
            return null;
        }

        $trimmed = trim($repo, " \t\n\r\0\x0B/");

        return $trimmed !== '' ? $trimmed : null;
    }

    private function buildForgeUrl(string $repo, string $sha): string
    {
        return 'https://forge.lthn.sh/'.$repo.'/commit/'.$sha;
    }

    private function profileName(?\Mod\AgentProfile $profile = null): string
    {
        $name = $profile?->name;

        if (is_string($name) && trim($name) !== '') {
            return trim($name);
        }

        return 'unknown';
    }
}
