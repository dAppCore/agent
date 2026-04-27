<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Support\Facades\Cache;
use Symfony\Component\Yaml\Yaml;

final class McpHealthService
{
    public const STATUS_ONLINE = 'online';

    public const STATUS_OFFLINE = 'offline';

    public const STATUS_DEGRADED = 'degraded';

    public const STATUS_UNKNOWN = 'unknown';

    protected int $cacheTtl = 60;

    protected int $timeout = 5;

    public function check(string $serverId, bool $forceRefresh = false): array
    {
        $cacheKey = sprintf('mcp:health:%s', $serverId);

        if (! $forceRefresh && Cache::has($cacheKey)) {
            return (array) Cache::get($cacheKey, []);
        }

        $server = $this->loadServerConfig($serverId);

        if ($server === null) {
            $result = $this->buildResult(self::STATUS_UNKNOWN, 'Server not found');
            Cache::put($cacheKey, $result, $this->cacheTtl);

            return $result;
        }

        $result = $this->pingServer($server);
        Cache::put($cacheKey, $result, $this->cacheTtl);

        return $result;
    }

    public function checkAll(bool $forceRefresh = false): array
    {
        $results = [];

        foreach ($this->registeredServers() as $serverId) {
            $results[$serverId] = $this->check($serverId, $forceRefresh);
        }

        return $results;
    }

    public function getCachedStatus(string $serverId): ?array
    {
        $status = Cache::get(sprintf('mcp:health:%s', $serverId));

        return is_array($status) ? $status : null;
    }

    public function clearCache(string $serverId): void
    {
        Cache::forget(sprintf('mcp:health:%s', $serverId));
    }

    public function clearAllCache(): void
    {
        foreach ($this->registeredServers() as $serverId) {
            $this->clearCache($serverId);
        }
    }

    public function getStatusBadge(string $status): string
    {
        return match ($status) {
            self::STATUS_ONLINE => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">Online</span>',
            self::STATUS_OFFLINE => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">Offline</span>',
            self::STATUS_DEGRADED => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">Degraded</span>',
            default => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">Unknown</span>',
        };
    }

    public function getStatusColour(string $status): string
    {
        return match ($status) {
            self::STATUS_ONLINE => 'green',
            self::STATUS_OFFLINE => 'red',
            self::STATUS_DEGRADED => 'yellow',
            default => 'gray',
        };
    }

    protected function pingServer(array $server): array
    {
        $connection = (array) ($server['connection'] ?? []);
        $type = (string) ($connection['type'] ?? 'stdio');

        if ($type !== 'stdio') {
            return $this->buildResult(self::STATUS_UNKNOWN, sprintf(
                "Connection type '%s' health check not supported",
                $type,
            ));
        }

        $command = trim($this->resolveEnvVars((string) ($connection['command'] ?? '')));
        if ($command === '') {
            return $this->buildResult(self::STATUS_OFFLINE, 'No command configured');
        }

        $args = array_map(
            fn (mixed $value): string => $this->resolveEnvVars((string) $value),
            (array) ($connection['args'] ?? []),
        );
        $cwd = $this->resolveEnvVars((string) ($connection['cwd'] ?? getcwd()));
        $payload = json_encode([
            'jsonrpc' => '2.0',
            'method' => 'initialize',
            'params' => [
                'protocolVersion' => '2024-11-05',
                'capabilities' => new \stdClass,
                'clientInfo' => [
                    'name' => 'mcp-health-check',
                    'version' => '1.0.0',
                ],
            ],
            'id' => 1,
        ], JSON_UNESCAPED_SLASHES);

        $result = $this->executeProcess(array_merge([$command], $args), $cwd, $payload.PHP_EOL);
        $duration = (int) ($result['response_time_ms'] ?? 0);
        $output = (string) ($result['output'] ?? '');
        $error = trim((string) ($result['error'] ?? ''));
        $exitCode = (int) ($result['exit_code'] ?? 1);

        if ($exitCode === 0 && $output !== '') {
            foreach (preg_split('/\R/', trim($output)) ?: [] as $line) {
                $decoded = json_decode($line, true);

                if (is_array($decoded) && isset($decoded['result'])) {
                    return $this->buildResult(self::STATUS_ONLINE, 'Server responding', [
                        'response_time_ms' => $duration,
                        'server_info' => $decoded['result']['serverInfo'] ?? null,
                        'protocol_version' => $decoded['result']['protocolVersion'] ?? null,
                    ]);
                }
            }

            return $this->buildResult(self::STATUS_DEGRADED, 'Server started but returned unexpected response', [
                'response_time_ms' => $duration,
                'output' => substr($output, 0, 500),
            ]);
        }

        return $this->buildResult(self::STATUS_OFFLINE, 'Server failed to start', [
            'response_time_ms' => $duration,
            'exit_code' => $exitCode,
            'error' => $error !== '' ? substr($error, 0, 500) : null,
        ]);
    }

    protected function executeProcess(array $command, string $cwd, string $input): array
    {
        $descriptors = [
            0 => ['pipe', 'r'],
            1 => ['pipe', 'w'],
            2 => ['pipe', 'w'],
        ];

        $startedAt = microtime(true);
        $process = @proc_open($command, $descriptors, $pipes, $cwd);

        if (! is_resource($process)) {
            return [
                'exit_code' => 1,
                'output' => '',
                'error' => 'Unable to start process',
                'response_time_ms' => 0,
            ];
        }

        fwrite($pipes[0], $input);
        fclose($pipes[0]);

        stream_set_blocking($pipes[1], false);
        stream_set_blocking($pipes[2], false);

        $stdout = '';
        $stderr = '';
        $timedOut = false;

        while (true) {
            $stdout .= stream_get_contents($pipes[1]);
            $stderr .= stream_get_contents($pipes[2]);
            $status = proc_get_status($process);

            if (! is_array($status) || ! ($status['running'] ?? false)) {
                break;
            }

            if ((microtime(true) - $startedAt) >= $this->timeout) {
                $timedOut = true;
                proc_terminate($process, 9);
                break;
            }

            usleep(100000);
        }

        $stdout .= stream_get_contents($pipes[1]);
        $stderr .= stream_get_contents($pipes[2]);

        fclose($pipes[1]);
        fclose($pipes[2]);

        $closeCode = proc_close($process);
        $exitCode = $timedOut ? 124 : $closeCode;

        return [
            'exit_code' => $exitCode,
            'output' => $stdout,
            'error' => $timedOut ? trim($stderr."\nTimed out waiting for MCP response.") : $stderr,
            'response_time_ms' => (int) round((microtime(true) - $startedAt) * 1000),
        ];
    }

    protected function buildResult(string $status, string $message, array $extra = []): array
    {
        return array_merge([
            'status' => $status,
            'message' => $message,
            'checked_at' => now()->toIso8601String(),
        ], array_filter($extra, static fn (mixed $value): bool => $value !== null));
    }

    protected function registeredServers(): array
    {
        $servers = $this->loadRegistry()['servers'] ?? [];

        return array_values(array_filter(array_map(
            static fn (mixed $server): ?string => is_array($server) && isset($server['id']) ? (string) $server['id'] : null,
            is_array($servers) ? $servers : [],
        )));
    }

    protected function loadRegistry(): array
    {
        $path = resource_path('mcp/registry.yaml');

        return file_exists($path) ? (array) Yaml::parseFile($path) : ['servers' => []];
    }

    protected function loadServerConfig(string $serverId): ?array
    {
        $path = resource_path(sprintf('mcp/servers/%s.yaml', $serverId));

        return file_exists($path) ? (array) Yaml::parseFile($path) : null;
    }

    protected function resolveEnvVars(string $value): string
    {
        return preg_replace_callback('/\$\{([^}]+)\}/', static function (array $matches): string {
            $parts = explode(':-', $matches[1], 2);
            $name = $parts[0];
            $default = $parts[1] ?? '';

            return (string) env($name, $default);
        }, $value) ?? $value;
    }
}
