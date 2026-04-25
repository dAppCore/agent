<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Symfony\Component\Yaml\Yaml;

final class OpenApiGenerator
{
    protected array $registry = ['servers' => []];

    protected array $servers = [];

    public function generate(): array
    {
        $this->loadRegistry();
        $this->loadServers();

        return [
            'openapi' => '3.0.3',
            'info' => $this->buildInfo(),
            'servers' => $this->buildServers(),
            'tags' => $this->buildTags(),
            'paths' => $this->buildPaths(),
            'components' => $this->buildComponents(),
        ];
    }

    public function toJson(): string
    {
        return (string) json_encode($this->generate(), JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    }

    public function toYaml(): string
    {
        return Yaml::dump($this->generate(), 10, 2);
    }

    protected function loadRegistry(): void
    {
        $path = resource_path('mcp/registry.yaml');
        $this->registry = file_exists($path) ? (array) Yaml::parseFile($path) : ['servers' => []];
    }

    protected function loadServers(): void
    {
        $this->servers = [];

        foreach ((array) ($this->registry['servers'] ?? []) as $reference) {
            if (! is_array($reference) || ! isset($reference['id'])) {
                continue;
            }

            $id = (string) $reference['id'];
            $path = resource_path(sprintf('mcp/servers/%s.yaml', $id));
            $this->servers[$id] = file_exists($path)
                ? (array) Yaml::parseFile($path)
                : ['id' => $id, 'name' => $id];
        }
    }

    protected function buildInfo(): array
    {
        return [
            'title' => 'Host UK MCP API',
            'description' => 'HTTP API for MCP server discovery, tool execution, and resource reads.',
            'version' => '1.0.0',
            'contact' => [
                'name' => 'Host UK Support',
                'url' => 'https://host.uk.com/contact',
                'email' => 'support@host.uk.com',
            ],
            'license' => [
                'name' => 'Proprietary',
            ],
        ];
    }

    protected function buildServers(): array
    {
        return [
            [
                'url' => 'https://mcp.host.uk.com/api/v1/mcp',
                'description' => 'Production',
            ],
            [
                'url' => 'https://mcp.test/api/v1/mcp',
                'description' => 'Local development',
            ],
        ];
    }

    protected function buildTags(): array
    {
        $tags = [
            ['name' => 'Discovery', 'description' => 'Server and tool discovery endpoints'],
            ['name' => 'Execution', 'description' => 'Tool execution and resource endpoints'],
        ];

        foreach ($this->servers as $server) {
            $tags[] = [
                'name' => (string) ($server['name'] ?? $server['id'] ?? 'unknown'),
                'description' => (string) ($server['tagline'] ?? $server['description'] ?? ''),
            ];
        }

        return $tags;
    }

    protected function buildPaths(): array
    {
        return [
            '/servers' => [
                'get' => [
                    'tags' => ['Discovery'],
                    'summary' => 'List all MCP servers',
                    'operationId' => 'listServers',
                    'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                    'responses' => [
                        '200' => [
                            'description' => 'List of available servers',
                            'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ServerList']]],
                        ],
                    ],
                ],
            ],
            '/servers/{serverId}' => [
                'get' => [
                    'tags' => ['Discovery'],
                    'summary' => 'Get server details',
                    'operationId' => 'getServer',
                    'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                    'parameters' => [[
                        'name' => 'serverId',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                    ]],
                    'responses' => [
                        '200' => [
                            'description' => 'Server details',
                            'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/Server']]],
                        ],
                        '404' => ['description' => 'Server not found'],
                    ],
                ],
            ],
            '/servers/{serverId}/tools' => [
                'get' => [
                    'tags' => ['Discovery'],
                    'summary' => 'List server tools',
                    'operationId' => 'listServerTools',
                    'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                    'parameters' => [[
                        'name' => 'serverId',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                    ]],
                    'responses' => [
                        '200' => [
                            'description' => 'Tool list',
                            'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ToolList']]],
                        ],
                    ],
                ],
            ],
            '/servers/{serverId}/resources' => [
                'get' => [
                    'tags' => ['Discovery'],
                    'summary' => 'List server resources',
                    'operationId' => 'listServerResources',
                    'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                    'parameters' => [[
                        'name' => 'serverId',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                    ]],
                    'responses' => [
                        '200' => [
                            'description' => 'Resource list',
                            'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ResourceList']]],
                        ],
                    ],
                ],
            ],
            '/tools/call' => [
                'post' => [
                    'tags' => ['Execution'],
                    'summary' => 'Execute an MCP tool',
                    'operationId' => 'callTool',
                    'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                    'requestBody' => [
                        'required' => true,
                        'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ToolCallRequest']]],
                    ],
                    'responses' => [
                        '200' => [
                            'description' => 'Tool executed successfully',
                            'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ToolCallResponse']]],
                        ],
                        '400' => ['description' => 'Invalid request'],
                        '401' => ['description' => 'Unauthorized'],
                        '404' => ['description' => 'Server or tool not found'],
                        '500' => ['description' => 'Tool execution error'],
                    ],
                ],
            ],
            '/resources/{uri}' => [
                'get' => [
                    'tags' => ['Execution'],
                    'summary' => 'Read a resource',
                    'operationId' => 'readResource',
                    'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                    'parameters' => [[
                        'name' => 'uri',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                    ]],
                    'responses' => [
                        '200' => [
                            'description' => 'Resource payload',
                            'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ResourceResponse']]],
                        ],
                    ],
                ],
            ],
        ];
    }

    protected function buildComponents(): array
    {
        return [
            'securitySchemes' => [
                'bearerAuth' => [
                    'type' => 'http',
                    'scheme' => 'bearer',
                    'description' => 'API key in bearer format, e.g. hk_xxx_yyy',
                ],
                'apiKeyAuth' => [
                    'type' => 'apiKey',
                    'in' => 'header',
                    'name' => 'X-API-Key',
                ],
            ],
            'schemas' => [
                'ServerList' => [
                    'type' => 'object',
                    'properties' => [
                        'servers' => ['type' => 'array', 'items' => ['$ref' => '#/components/schemas/ServerSummary']],
                        'count' => ['type' => 'integer'],
                    ],
                ],
                'ServerSummary' => [
                    'type' => 'object',
                    'properties' => [
                        'id' => ['type' => 'string'],
                        'name' => ['type' => 'string'],
                        'tagline' => ['type' => 'string'],
                        'tool_count' => ['type' => 'integer'],
                        'resource_count' => ['type' => 'integer'],
                    ],
                ],
                'Server' => [
                    'type' => 'object',
                    'properties' => [
                        'id' => ['type' => 'string'],
                        'name' => ['type' => 'string'],
                        'tagline' => ['type' => 'string'],
                        'description' => ['type' => 'string'],
                        'tools' => ['type' => 'array', 'items' => ['$ref' => '#/components/schemas/Tool']],
                        'resources' => ['type' => 'array', 'items' => ['$ref' => '#/components/schemas/Resource']],
                    ],
                ],
                'Tool' => [
                    'type' => 'object',
                    'properties' => [
                        'name' => ['type' => 'string'],
                        'description' => ['type' => 'string'],
                        'inputSchema' => ['type' => 'object', 'additionalProperties' => true],
                    ],
                ],
                'Resource' => [
                    'type' => 'object',
                    'properties' => [
                        'uri' => ['type' => 'string'],
                        'name' => ['type' => 'string'],
                        'description' => ['type' => 'string'],
                        'mimeType' => ['type' => 'string'],
                    ],
                ],
                'ToolList' => [
                    'type' => 'object',
                    'properties' => [
                        'server' => ['type' => 'string'],
                        'tools' => ['type' => 'array', 'items' => ['$ref' => '#/components/schemas/Tool']],
                        'count' => ['type' => 'integer'],
                    ],
                ],
                'ToolCallRequest' => [
                    'type' => 'object',
                    'required' => ['server', 'tool'],
                    'properties' => [
                        'server' => ['type' => 'string'],
                        'tool' => ['type' => 'string'],
                        'arguments' => ['type' => 'object', 'additionalProperties' => true],
                    ],
                ],
                'ToolCallResponse' => [
                    'type' => 'object',
                    'properties' => [
                        'success' => ['type' => 'boolean'],
                        'server' => ['type' => 'string'],
                        'tool' => ['type' => 'string'],
                        'result' => ['type' => 'object', 'additionalProperties' => true],
                        'duration_ms' => ['type' => 'integer'],
                        'error' => ['type' => 'string'],
                    ],
                ],
                'ResourceResponse' => [
                    'type' => 'object',
                    'properties' => [
                        'uri' => ['type' => 'string'],
                        'content' => ['type' => 'object', 'additionalProperties' => true],
                    ],
                ],
                'ResourceList' => [
                    'type' => 'object',
                    'properties' => [
                        'server' => ['type' => 'string'],
                        'resources' => ['type' => 'array', 'items' => ['$ref' => '#/components/schemas/Resource']],
                        'count' => ['type' => 'integer'],
                    ],
                ],
            ],
        ];
    }
}
