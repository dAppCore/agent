<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Search;

use Core\Mod\Agentic\Mod\Admin\Search\Contracts\SearchProvider;
use Illuminate\Support\Str;

class SearchProviderRegistry
{
    /**
     * @var array<int, SearchProvider>
     */
    protected array $providers = [];

    public function register(SearchProvider $provider): void
    {
        $this->providers[] = $provider;
    }

    /**
     * @param  array<int, SearchProvider>  $providers
     */
    public function registerMany(array $providers): void
    {
        foreach ($providers as $provider) {
            $this->register($provider);
        }
    }

    /**
     * @return array<int, SearchProvider>
     */
    public function providers(): array
    {
        return $this->providers;
    }

    /**
     * @return array<string, array{label: string, icon: string, results: array<int, array<string, mixed>>}>
     */
    public function search(string $query, ?object $user = null, ?object $workspace = null, int $limitPerProvider = 5): array
    {
        $grouped = [];

        foreach ($this->availableProviders($user, $workspace) as $provider) {
            $results = array_slice($provider->search($query), 0, $limitPerProvider);

            if ($results === []) {
                continue;
            }

            $key = Str::slug($provider->name(), '_');
            $grouped[$key] = [
                'label' => $provider->name(),
                'icon' => $provider->icon(),
                'results' => array_map(
                    static fn (mixed $result): array => $result instanceof SearchResult
                        ? $result->toArray()
                        : SearchResult::fromArray((array) $result)->toArray(),
                    $results
                ),
            ];
        }

        return $grouped;
    }

    /**
     * @return array<int, SearchProvider>
     */
    protected function availableProviders(?object $user = null, ?object $workspace = null): array
    {
        $providers = array_filter($this->providers, static function (SearchProvider $provider) use ($user, $workspace): bool {
            if (! method_exists($provider, 'available')) {
                return true;
            }

            return (bool) $provider->available($user, $workspace);
        });

        usort($providers, static function (SearchProvider $left, SearchProvider $right): int {
            $leftPriority = method_exists($left, 'priority') ? (int) $left->priority() : 50;
            $rightPriority = method_exists($right, 'priority') ? (int) $right->priority() : 50;

            return $leftPriority <=> $rightPriority;
        });

        return array_values($providers);
    }

    /**
     * @param  array<string, array{label: string, icon: string, results: array<int, array<string, mixed>>}>  $grouped
     * @return array<int, array<string, mixed>>
     */
    public function flattenResults(array $grouped): array
    {
        $flat = [];

        foreach ($grouped as $group) {
            foreach ($group['results'] as $result) {
                $flat[] = $result;
            }
        }

        return $flat;
    }

    public function fuzzyMatch(string $query, string $target): bool
    {
        $query = Str::lower(trim($query));
        $target = Str::lower(trim($target));

        if ($query === '') {
            return false;
        }

        if (Str::contains($target, $query)) {
            return true;
        }

        $words = preg_split('/\s+/', $target) ?: [];
        $queryChars = str_split($query);
        $wordIndex = 0;
        $charIndex = 0;

        while ($charIndex < count($queryChars) && $wordIndex < count($words)) {
            if (Str::startsWith($words[$wordIndex], $queryChars[$charIndex])) {
                $charIndex++;
            }
            $wordIndex++;
        }

        if ($charIndex === count($queryChars)) {
            return true;
        }

        $targetIndex = 0;

        foreach ($queryChars as $char) {
            $foundAt = strpos($target, $char, $targetIndex);
            if ($foundAt === false) {
                return false;
            }
            $targetIndex = $foundAt + 1;
        }

        return true;
    }

    public function relevanceScore(string $query, string $target): int
    {
        $query = Str::lower(trim($query));
        $target = Str::lower(trim($target));

        if ($query === '' || $target === '') {
            return 0;
        }

        if ($query === $target) {
            return 100;
        }

        if (Str::startsWith($target, $query)) {
            return 90;
        }

        if (Str::contains($target, $query)) {
            return 70;
        }

        return $this->fuzzyMatch($query, $target) ? 60 : 0;
    }
}
