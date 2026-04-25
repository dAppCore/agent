<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Console;

use Core\Mod\Agentic\Mcp\Services\McpQuotaService;
use Core\Mod\Agentic\Mcp\Services\QueryAuditService;
use Core\Mod\Agentic\Mcp\Services\ToolRegistry;
use Illuminate\Console\Command;
use InvalidArgumentException;
use JsonException;
use RuntimeException;
use Throwable;

class McpAgentServerCommand extends Command
{
    protected $signature = 'mcp:agent-server';

    protected $description = 'Run the MCP agent server in stdio mode';

    public function handle(
        ToolRegistry $toolRegistry,
        McpQuotaService $quotaService,
        QueryAuditService $queryAuditService,
    ): int {
        $inputPath = $this->streamPath('MCP_AGENT_SERVER_INPUT', 'php://stdin');
        $outputPath = $this->streamPath('MCP_AGENT_SERVER_OUTPUT', 'php://stdout');
        $input = @fopen($inputPath, 'rb');
        $output = @fopen($outputPath, 'wb');

        if (! is_resource($input) || ! is_resource($output)) {
            $this->error('Unable to open MCP stdio streams.');

            return self::FAILURE;
        }

        $reachedEof = false;

        try {
            while (($line = fgets($input)) !== false) {
                $payload = trim($line);

                if ($payload === '') {
                    continue;
                }

                $response = $this->processPayload(
                    $payload,
                    $toolRegistry,
                    $quotaService,
                    $queryAuditService,
                );

                if ($response === null) {
                    continue;
                }

                fwrite($output, $this->encodePayload($response).PHP_EOL);
                fflush($output);
            }

            $reachedEof = feof($input);
        } finally {
            if ($inputPath !== 'php://stdin' && is_resource($input)) {
                fclose($input);
            }

            if ($outputPath !== 'php://stdout' && is_resource($output)) {
                fclose($output);
            }
        }

        if (! $reachedEof) {
            $this->error('MCP stdio stream closed unexpectedly.');

            return self::FAILURE;
        }

        return self::SUCCESS;
    }

    private function streamPath(string $variable, string $default): string
    {
        $value = getenv($variable);

        return is_string($value) && $value !== '' ? $value : $default;
    }

    private function processPayload(
        string $payload,
        ToolRegistry $toolRegistry,
        McpQuotaService $quotaService,
        QueryAuditService $queryAuditService,
    ): array|null {
        try {
            $request = json_decode($payload, true, 512, JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            return $this->errorResponse(-32700, 'Parse error');
        }

        if (! is_array($request)) {
            return $this->errorResponse(-32600, 'Invalid Request');
        }

        if (array_is_list($request)) {
            if ($request === []) {
                return $this->errorResponse(-32600, 'Invalid Request');
            }

            $responses = [];

            foreach ($request as $item) {
                $response = $this->processRequest(
                    is_array($item) ? $item : [],
                    $toolRegistry,
                    $quotaService,
                    $queryAuditService,
                );

                if ($response !== null) {
                    $responses[] = $response;
                }
            }

            return $responses === [] ? null : $responses;
        }

        return $this->processRequest(
            $request,
            $toolRegistry,
            $quotaService,
            $queryAuditService,
        );
    }

    private function processRequest(
        array $request,
        ToolRegistry $toolRegistry,
        McpQuotaService $quotaService,
        QueryAuditService $queryAuditService,
    ): array|null {
        $id = $request['id'] ?? null;

        if (($request['jsonrpc'] ?? null) !== '2.0') {
            return $this->errorResponse(-32600, 'Invalid Request', $id);
        }

        $method = $request['method'] ?? null;
        if (! is_string($method) || $method === '') {
            return $this->errorResponse(-32600, 'Invalid Request', $id);
        }

        $params = $request['params'] ?? [];
        if (! is_array($params)) {
            return $this->errorResponse(-32602, 'Invalid params', $id);
        }

        $response = match ($method) {
            'initialize' => $this->successResponse([
                'serverInfo' => [
                    'name' => 'core-agent-mcp',
                    'transport' => 'stdio',
                ],
                'capabilities' => [
                    'tools' => [
                        'list' => true,
                        'call' => true,
                    ],
                ],
            ], $id),
            'ping' => $this->successResponse(['ok' => true], $id),
            'tools/list' => $this->successResponse([
                'tools' => array_map(
                    static fn ($tool): array => [
                        'name' => $tool->name,
                        'description' => $tool->description,
                        'inputSchema' => $tool->inputSchema,
                        'dependencies' => $tool->dependencies,
                        'metadata' => $tool->metadata,
                    ],
                    $toolRegistry->listTools(),
                ),
            ], $id),
            'tools/call' => $this->handleToolCall(
                $params,
                $id,
                $toolRegistry,
                $quotaService,
                $queryAuditService,
            ),
            'resources/list' => $this->successResponse(['resources' => []], $id),
            default => $this->errorResponse(-32601, 'Method not found', $id),
        };

        if (! array_key_exists('id', $request)) {
            return null;
        }

        return $response;
    }

    private function handleToolCall(
        array $params,
        mixed $id,
        ToolRegistry $toolRegistry,
        McpQuotaService $quotaService,
        QueryAuditService $queryAuditService,
    ): array {
        $toolName = $this->resolveToolName($params);
        if ($toolName === null) {
            return $this->errorResponse(-32602, 'Tool name is required.', $id);
        }

        $arguments = $params['arguments'] ?? $params['input'] ?? [];
        if (! is_array($arguments)) {
            return $this->errorResponse(-32602, 'Tool arguments must be an array.', $id);
        }

        $workspaceId = $this->workspaceId($params, $arguments);
        $quota = $quotaService->checkQuota($workspaceId);

        if ($quota->exceeded) {
            return $this->errorResponse(-32001, 'MCP quota exceeded.', $id, [
                'quota' => $quota->toArray(),
                'workspace_id' => $workspaceId,
            ]);
        }

        $query = $this->toolQuery($arguments);
        if ($query !== null && ! $queryAuditService->isSafe($query)) {
            $this->recordAudit(
                $queryAuditService,
                $query,
                $workspaceId,
                $toolName,
                0,
                ['rejected' => true],
            );

            return $this->errorResponse(-32002, 'Unsafe MCP query rejected.', $id, [
                'tool' => $toolName,
                'workspace_id' => $workspaceId,
            ]);
        }

        $consumedQuota = $quotaService->consume($workspaceId);
        $startedAt = microtime(true);

        try {
            $result = $toolRegistry->call($toolName, $arguments, [
                'workspace_id' => $workspaceId,
                'request_id' => $id,
                'transport' => 'stdio',
            ]);
            $durationMs = (int) round((microtime(true) - $startedAt) * 1000);

            if ($query !== null) {
                $this->recordAudit(
                    $queryAuditService,
                    $query,
                    $workspaceId,
                    $toolName,
                    $durationMs,
                    ['request_id' => $id, 'transport' => 'stdio'],
                    $this->resultCount($result),
                );
            }

            return $this->successResponse([
                'tool' => $toolName,
                'result' => $result,
                'quota' => $consumedQuota->toArray(),
                'duration_ms' => $durationMs,
            ], $id);
        } catch (InvalidArgumentException $exception) {
            return $this->errorResponse(-32602, $exception->getMessage(), $id, [
                'tool' => $toolName,
            ]);
        } catch (Throwable $exception) {
            $durationMs = (int) round((microtime(true) - $startedAt) * 1000);

            if ($query !== null) {
                $this->recordAudit(
                    $queryAuditService,
                    $query,
                    $workspaceId,
                    $toolName,
                    $durationMs,
                    [
                        'request_id' => $id,
                        'transport' => 'stdio',
                        'error' => $exception->getMessage(),
                    ],
                );
            }

            return $this->errorResponse(-32603, $exception->getMessage(), $id, [
                'tool' => $toolName,
                'workspace_id' => $workspaceId,
            ]);
        }
    }

    private function resolveToolName(array $params): string|null
    {
        $value = $params['name'] ?? $params['tool'] ?? null;

        return is_string($value) && $value !== '' ? $value : null;
    }

    private function workspaceId(array $params, array $arguments): string
    {
        $value = $params['workspace_id']
            ?? $params['workspace']
            ?? $arguments['workspace_id']
            ?? $arguments['workspace']
            ?? 'default';

        return (string) $value;
    }

    private function toolQuery(array $arguments): string|null
    {
        $query = $arguments['query'] ?? null;

        return is_string($query) && $query !== '' ? $query : null;
    }

    private function resultCount(mixed $result): int
    {
        if (is_array($result)) {
            return count($result);
        }

        return $result === null ? 0 : 1;
    }

    private function recordAudit(
        QueryAuditService $queryAuditService,
        string $query,
        string $workspaceId,
        string $toolName,
        int $durationMs,
        array $metadata = [],
        ?int $resultCount = null,
    ): void {
        try {
            $queryAuditService->log($query, [
                'workspace_id' => $workspaceId,
                'tool_name' => $toolName,
                'result_count' => $resultCount,
                'duration_ms' => $durationMs,
                'metadata' => $metadata,
            ]);
        } catch (RuntimeException) {
            // The audit table is optional in this worktree.
        }
    }

    private function successResponse(mixed $result, mixed $id): array
    {
        return [
            'jsonrpc' => '2.0',
            'result' => $result,
            'id' => $id,
        ];
    }

    private function errorResponse(int $code, string $message, mixed $id = null, array $data = []): array
    {
        $error = [
            'code' => $code,
            'message' => $message,
        ];

        if ($data !== []) {
            $error['data'] = $data;
        }

        return [
            'jsonrpc' => '2.0',
            'error' => $error,
            'id' => $id,
        ];
    }

    private function encodePayload(mixed $payload): string
    {
        $encoded = json_encode(
            $payload,
            JSON_INVALID_UTF8_SUBSTITUTE | JSON_UNESCAPED_SLASHES,
        );

        if ($encoded === false) {
            return '{"jsonrpc":"2.0","error":{"code":-32603,"message":"Unable to encode response payload."},"id":null}';
        }

        return $encoded;
    }
}
