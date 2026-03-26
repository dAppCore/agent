<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Contracts;

/**
 * Contract for MCP Agent Server tools.
 *
 * Tools extracted from the monolithic McpAgentServerCommand
 * implement this interface for clean separation of concerns.
 */
interface AgentToolInterface
{
    /**
     * Get the tool name (used as the MCP tool identifier).
     */
    public function name(): string;

    /**
     * Get the tool description for MCP clients.
     */
    public function description(): string;

    /**
     * Get the JSON Schema for tool input parameters.
     */
    public function inputSchema(): array;

    /**
     * Execute the tool with the given arguments.
     *
     * @param  array  $args  Input arguments from MCP client
     * @param  array  $context  Execution context (session_id, workspace_id, etc.)
     * @return array Tool result
     */
    public function handle(array $args, array $context = []): array;

    /**
     * Get required permission scopes to execute this tool.
     *
     * @return array<string> List of required scopes
     */
    public function requiredScopes(): array;

    /**
     * Get the tool category for grouping.
     */
    public function category(): string;
}
