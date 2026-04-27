<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Symfony\Component\Yaml\Exception\ParseException;
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
        if (! file_exists($path)) {
            $this->registry = ['servers' => []];

            return;
        }

        try {
            $this->registry = (array) Yaml::parseFile($path);
        } catch (ParseException) {
            $this->registry = ['servers' => []];
        }
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
            if (! file_exists($path)) {
                $this->servers[$id] = ['id' => $id, 'name' => $id];

                continue;
            }

            try {
                $this->servers[$id] = (array) Yaml::parseFile($path);
            } catch (ParseException) {
                $this->servers[$id] = ['id' => $id, 'name' => $id];
            }
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
                'get' => $this->authenticatedGet(
                    'Discovery',
                    'List all MCP servers',
                    'listServers',
                    [
                        '200' => $this->schemaResponse(
                            'List of available servers',
                            '#/components/schemas/ServerList',
                        ),
                    ],
                ),
            ],
            '/servers/{serverId}' => [
                'get' => $this->authenticatedGet(
                    'Discovery',
                    'Get server details',
                    'getServer',
                    [
                        '200' => $this->schemaResponse(
                            'Server details',
                            '#/components/schemas/Server',
                        ),
                        '404' => ['description' => 'Server not found'],
                    ],
                    [$this->requiredStringParameter('serverId', 'path')],
                ),
            ],
            '/servers/{serverId}/tools' => [
                'get' => $this->authenticatedGet(
                    'Discovery',
                    'List server tools',
                    'listServerTools',
                    [
                        '200' => $this->schemaResponse(
                            'Tool list',
                            '#/components/schemas/ToolList',
                        ),
                    ],
                    [$this->requiredStringParameter('serverId', 'path')],
                ),
            ],
            '/servers/{serverId}/resources' => [
                'get' => $this->authenticatedGet(
                    'Discovery',
                    'List server resources',
                    'listServerResources',
                    [
                        '200' => $this->schemaResponse(
                            'Resource list',
                            '#/components/schemas/ResourceList',
                        ),
                    ],
                    [$this->requiredStringParameter('serverId', 'path')],
                ),
            ],
            '/tools/call' => [
                'post' => $this->authenticatedPost(
                    'Execution',
                    'Execute an MCP tool',
                    'callTool',
                    '#/components/schemas/ToolCallRequest',
                    [
                        '200' => $this->schemaResponse(
                            'Tool executed successfully',
                            '#/components/schemas/ToolCallResponse',
                        ),
                        '400' => ['description' => 'Invalid request'],
                        '401' => ['description' => 'Unauthorized'],
                        '404' => ['description' => 'Server or tool not found'],
                        '500' => ['description' => 'Tool execution error'],
                    ],
                ),
            ],
            '/resources' => [
                'get' => $this->authenticatedGet(
                    'Execution',
                    'Read a resource',
                    'readResource',
                    [
                        '200' => $this->schemaResponse(
                            'Resource payload',
                            '#/components/schemas/ResourceResponse',
                        ),
                    ],
                    [$this->requiredStringParameter('uri', 'query')],
                ),
            ],
        ];
    }

    protected function authenticatedGet(
        string $tag,
        string $summary,
        string $operationId,
        array $responses,
        array $parameters = [],
    ): array {
        $operation = [
            'tags' => [$tag],
            'summary' => $summary,
            'operationId' => $operationId,
            'security' => $this->securityRequirements(),
            'responses' => $responses,
        ];

        if ($parameters !== []) {
            $operation['parameters'] = $parameters;
        }

        return $operation;
    }

    protected function authenticatedPost(
        string $tag,
        string $summary,
        string $operationId,
        string $requestSchemaRef,
        array $responses,
    ): array {
        return [
            'tags' => [$tag],
            'summary' => $summary,
            'operationId' => $operationId,
            'security' => $this->securityRequirements(),
            'requestBody' => [
                'required' => true,
                'content' => $this->jsonSchemaContent($requestSchemaRef),
            ],
            'responses' => $responses,
        ];
    }

    protected function securityRequirements(): array
    {
        return [['bearerAuth' => []], ['apiKeyAuth' => []]];
    }

    protected function requiredStringParameter(string $name, string $location): array
    {
        return [
            'name' => $name,
            'in' => $location,
            'required' => true,
            'schema' => ['type' => 'string'],
        ];
    }

    protected function schemaResponse(string $description, string $schemaRef): array
    {
        return [
            'description' => $description,
            'content' => $this->jsonSchemaContent($schemaRef),
        ];
    }

    protected function jsonSchemaContent(string $schemaRef): array
    {
        return [
            'application/json' => [
                'schema' => ['$ref' => $schemaRef],
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
