<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Console\Command;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Storage;
use Symfony\Component\Finder\Finder;

class AgenticSyncPluginsCcCommand extends Command
{
    private const STORAGE_DISK = 'agentic_plugins_cc';

    protected $signature = 'agentic:sync-plugins-cc';

    protected $description = 'Synchronise Claude Code plugins against enabled agent profiles';

    public function handle(): int
    {
        try {
            $pluginsPath = $this->pluginsPath();
        } catch (\RuntimeException $exception) {
            $this->error($exception->getMessage());

            return self::FAILURE;
        }

        $pluginNames = $this->discoverPluginNames($pluginsPath);

        if ($pluginNames === []) {
            $this->info("No Claude Code plugin directories found in {$pluginsPath}.");

            return self::SUCCESS;
        }

        $profiles = AgentProfile::query()
            ->where('enabled', true)
            ->get();

        $claimedProfileIds = [];
        $report = [];
        $pendingPluginNames = [];

        foreach ($pluginNames as $pluginName) {
            $nameMatches = $this->matchProfilesByName($profiles, $pluginName, $claimedProfileIds);

            if ($nameMatches->count() === 1) {
                $profile = $nameMatches->first();
                $claimedProfileIds[$profile->id] = $pluginName;
                $report[] = $this->mapPluginToProfile($pluginName, $profile, 'name');

                continue;
            }

            if ($nameMatches->count() > 1) {
                $pendingPluginNames[$pluginName] = 'ambiguous name match';
            } else {
                $pendingPluginNames[$pluginName] = 'no enabled profile';
            }
        }

        foreach ($pendingPluginNames as $pluginName => $fallbackReason) {
            $pluginNameMatches = $this->matchProfilesByPluginCcName($profiles, $pluginName, $claimedProfileIds);

            if ($pluginNameMatches->count() === 1) {
                $profile = $pluginNameMatches->first();
                $claimedProfileIds[$profile->id] = $pluginName;
                $report[] = $this->mapPluginToProfile($pluginName, $profile, 'plugin_cc_name');

                continue;
            }

            if ($pluginNameMatches->count() > 1) {
                $report[] = $this->unmappedRow($pluginName, 'ambiguous plugin_cc_name match');

                continue;
            }

            $report[] = $this->unmappedRow($pluginName, $fallbackReason);
        }

        usort($report, static fn (array $left, array $right): int => strcmp((string) $left['plugin'], (string) $right['plugin']));

        $this->table(
            ['Plugin', 'Status', 'Profile', 'Match', 'Action'],
            array_map(
                static fn (array $row): array => [
                    $row['plugin'],
                    $row['status'],
                    $row['profile'],
                    $row['match'],
                    $row['action'],
                ],
                $report,
            ),
        );

        $mapped = count(array_filter($report, static fn (array $row): bool => $row['status'] === 'mapped'));
        $unmapped = count($report) - $mapped;

        $this->info("Mapped {$mapped} Claude Code plugin(s); {$unmapped} unmapped.");

        return self::SUCCESS;
    }

    /**
     * @return array<int, string>
     */
    private function discoverPluginNames(string $pluginsPath): array
    {
        if (! is_dir($pluginsPath)) {
            return [];
        }

        $pluginNames = $this->discoverPluginNamesViaStorage($pluginsPath);

        if ($pluginNames === null) {
            $pluginNames = $this->discoverPluginNamesViaFinder($pluginsPath);
        }

        sort($pluginNames, SORT_NATURAL | SORT_FLAG_CASE);

        return array_values(array_unique($pluginNames));
    }

    /**
     * @return array<int, string>|null
     */
    private function discoverPluginNamesViaStorage(string $pluginsPath): ?array
    {
        try {
            config([
                'filesystems.disks.'.self::STORAGE_DISK => [
                    'driver' => 'local',
                    'root' => $pluginsPath,
                ],
            ]);

            $disk = Storage::disk(self::STORAGE_DISK);
            $directories = $disk->directories('/');
            $pluginNames = [];

            foreach ($directories as $directory) {
                $directory = trim($directory, '/');

                if ($directory === '') {
                    continue;
                }

                if (! $disk->exists($directory.'/plugin.json')) {
                    continue;
                }

                $pluginNames[] = basename($directory);
            }

            return $pluginNames;
        } catch (\Throwable) {
            return null;
        }
    }

    /**
     * @return array<int, string>
     */
    private function discoverPluginNamesViaFinder(string $pluginsPath): array
    {
        $finder = Finder::create()
            ->directories()
            ->depth('== 0')
            ->in($pluginsPath)
            ->sortByName();

        $pluginNames = [];

        foreach ($finder as $directory) {
            $pluginJsonPath = $directory->getRealPath().'/plugin.json';

            if (! is_file($pluginJsonPath)) {
                continue;
            }

            $pluginNames[] = $directory->getFilename();
        }

        return $pluginNames;
    }

    /**
     * @param array<int, string> $claimedProfileIds
     * @return Collection<int, AgentProfile>
     */
    private function matchProfilesByName(Collection $profiles, string $pluginName, array $claimedProfileIds): Collection
    {
        $normalisedPluginName = $this->normalise($pluginName);

        return $profiles
            ->reject(static fn (AgentProfile $profile): bool => array_key_exists($profile->id, $claimedProfileIds))
            ->filter(fn (AgentProfile $profile): bool => $this->normalise($profile->name) === $normalisedPluginName)
            ->values();
    }

    /**
     * @param array<int, string> $claimedProfileIds
     * @return Collection<int, AgentProfile>
     */
    private function matchProfilesByPluginCcName(Collection $profiles, string $pluginName, array $claimedProfileIds): Collection
    {
        $normalisedPluginName = $this->normalise($pluginName);

        return $profiles
            ->reject(static fn (AgentProfile $profile): bool => array_key_exists($profile->id, $claimedProfileIds))
            ->filter(function (AgentProfile $profile) use ($normalisedPluginName): bool {
                if (! is_string($profile->plugin_cc_name) || $profile->plugin_cc_name === '') {
                    return false;
                }

                return $this->normalise($profile->plugin_cc_name) === $normalisedPluginName;
            })
            ->values();
    }

    /**
     * @return array{plugin: string, status: string, profile: string, match: string, action: string}
     */
    private function mapPluginToProfile(string $pluginName, AgentProfile $profile, string $match): array
    {
        $action = $profile->plugin_cc_name === $pluginName ? 'unchanged' : 'updated';

        if ($action === 'updated') {
            $profile->forceFill([
                'plugin_cc_name' => $pluginName,
            ])->save();
        }

        return [
            'plugin' => $pluginName,
            'status' => 'mapped',
            'profile' => $profile->name,
            'match' => $match,
            'action' => $action,
        ];
    }

    /**
     * @return array{plugin: string, status: string, profile: string, match: string, action: string}
     */
    private function unmappedRow(string $pluginName, string $reason): array
    {
        return [
            'plugin' => $pluginName,
            'status' => 'unmapped',
            'profile' => '-',
            'match' => '-',
            'action' => $reason,
        ];
    }

    private function pluginsPath(): string
    {
        $home = getenv('HOME');

        if (! is_string($home) || $home === '') {
            $home = $_SERVER['HOME'] ?? $_ENV['HOME'] ?? '';
        }

        if (! is_string($home) || $home === '') {
            throw new \RuntimeException('Unable to resolve HOME for Claude Code plugin discovery.');
        }

        return rtrim($home, '/').'/.claude/plugins';
    }

    private function normalise(string $value): string
    {
        return strtolower(trim($value));
    }
}
