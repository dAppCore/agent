<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/McpHealthService.php');

use Core\Mcp\Services\McpHealthService;
use Illuminate\Support\Facades\Cache;

beforeEach(function (): void {
    Cache::flush();
});

test('McpHealthService_check_Good_marks_a_stdio_server_online_when_initialize_returns_jsonrpc_result', function (): void {
    $service = new class extends McpHealthService
    {
        protected function loadServerConfig(string $serverId): ?array
        {
            return [
                'id' => $serverId,
                'connection' => [
                    'type' => 'stdio',
                    'command' => 'php',
                    'args' => ['artisan', 'mcp:agent-server'],
                    'cwd' => '${APP_ROOT:-/tmp}',
                ],
            ];
        }

        protected function executeProcess(array $command, string $cwd, string $input): array
        {
            return [
                'exit_code' => 0,
                'output' => json_encode([
                    'jsonrpc' => '2.0',
                    'result' => [
                        'serverInfo' => ['name' => 'Host Hub'],
                        'protocolVersion' => '2024-11-05',
                    ],
                ]),
                'error' => '',
                'response_time_ms' => 42,
            ];
        }
    };

    $result = $service->check('host-hub', true);

    expect($result['status'])->toBe(McpHealthService::STATUS_ONLINE)
        ->and($result['protocol_version'])->toBe('2024-11-05')
        ->and($result['response_time_ms'])->toBe(42);
});

test('McpHealthService_check_Bad_returns_unknown_for_missing_registry_entries', function (): void {
    $service = new class extends McpHealthService
    {
        protected function loadServerConfig(string $serverId): ?array
        {
            return null;
        }
    };

    expect($service->check('missing', true)['status'])->toBe(McpHealthService::STATUS_UNKNOWN);
});

test('McpHealthService_check_Ugly_marks_successful_but_malformed_stdio_output_as_degraded', function (): void {
    $service = new class extends McpHealthService
    {
        protected function loadServerConfig(string $serverId): ?array
        {
            return [
                'id' => $serverId,
                'connection' => [
                    'type' => 'stdio',
                    'command' => 'php',
                ],
            ];
        }

        protected function executeProcess(array $command, string $cwd, string $input): array
        {
            return [
                'exit_code' => 0,
                'output' => "not-json\nstill-not-json",
                'error' => '',
                'response_time_ms' => 15,
            ];
        }
    };

    $result = $service->check('marketing', true);

    expect($result['status'])->toBe(McpHealthService::STATUS_DEGRADED)
        ->and($result['output'])->toContain('not-json');
});
