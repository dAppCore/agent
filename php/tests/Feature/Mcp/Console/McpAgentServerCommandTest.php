<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Console\McpAgentServerCommand;
use Core\Mod\Agentic\Mcp\Services\McpQuotaService;
use Core\Mod\Agentic\Mcp\Tools\Agent\Contracts\AgentToolInterface;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Services\AgentToolRegistry;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

beforeEach(function (): void {
    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(McpAgentServerCommand::class),
    );

    $this->app->make(AgentToolRegistry::class)->register(new class implements AgentToolInterface
    {
        public function name(): string
        {
            return 'echo_tool';
        }

        public function description(): string
        {
            return 'Echoes arguments back to the caller.';
        }

        public function inputSchema(): array
        {
            return ['type' => 'object'];
        }

        public function handle(array $arguments, array $context = []): array
        {
            return [
                'arguments' => $arguments,
                'context' => $context,
                'value' => $arguments['value'] ?? null,
            ];
        }

        public function requiredScopes(): array
        {
            return ['read'];
        }

        public function category(): string
        {
            return 'testing';
        }
    });

    Schema::dropIfExists('mcp_audit_entries');
    Schema::create('mcp_audit_entries', function (Blueprint $table): void {
        $table->id();
        $table->string('workspace_id')->nullable();
        $table->string('tool_name')->nullable();
        $table->longText('query_text');
        $table->string('query_hash', 64);
        $table->boolean('is_safe')->default(true);
        $table->unsignedInteger('result_count')->nullable();
        $table->unsignedInteger('duration_ms')->nullable();
        $table->json('metadata')->nullable();
        $table->timestamps();
    });
});

afterEach(function (): void {
    putenv('MCP_AGENT_SERVER_INPUT');
    putenv('MCP_AGENT_SERVER_OUTPUT');
});

function mcpAgentServerRun(string $request): array
{
    $inputPath = tempnam(sys_get_temp_dir(), 'mcp-agent-input-');
    $outputPath = tempnam(sys_get_temp_dir(), 'mcp-agent-output-');

    file_put_contents($inputPath, $request);
    putenv(sprintf('MCP_AGENT_SERVER_INPUT=%s', $inputPath));
    putenv(sprintf('MCP_AGENT_SERVER_OUTPUT=%s', $outputPath));

    $exitCode = Artisan::call('mcp:agent-server');
    $lines = file($outputPath, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) ?: [];
    $responses = array_map(
        static fn (string $line): array => json_decode($line, true, 512, JSON_THROW_ON_ERROR),
        $lines,
    );

    @unlink($inputPath);
    @unlink($outputPath);

    return [
        'exitCode' => $exitCode,
        'responses' => $responses,
    ];
}

test('McpAgentServerCommand_handle_Good_processes_stdio_tool_calls_and_records_safe_queries', function (): void {
    config()->set('mcp.quota_limit', 5);

    $result = mcpAgentServerRun(json_encode([
        'jsonrpc' => '2.0',
        'method' => 'tools/call',
        'params' => [
            'name' => 'echo_tool',
            'arguments' => [
                'workspace_id' => 'workspace-good',
                'query' => 'select * from agent_plans',
                'value' => 'pong',
            ],
        ],
        'id' => 7,
    ]).PHP_EOL);

    expect($result['exitCode'])->toBe(0)
        ->and($result['responses'])->toHaveCount(1)
        ->and($result['responses'][0]['result']['tool'])->toBe('echo_tool')
        ->and($result['responses'][0]['result']['result']['value'])->toBe('pong')
        ->and($result['responses'][0]['result']['quota']['used'])->toBe(1)
        ->and(Schema::hasTable('mcp_audit_entries'))->toBeTrue()
        ->and(DB::table('mcp_audit_entries')->count())->toBe(1);
});

test('McpAgentServerCommand_handle_Bad_returns_parse_errors_for_invalid_json', function (): void {
    $result = mcpAgentServerRun("{bad json\n");

    expect($result['exitCode'])->toBe(0)
        ->and($result['responses'])->toHaveCount(1)
        ->and($result['responses'][0]['error']['code'])->toBe(-32700)
        ->and($result['responses'][0]['error']['message'])->toBe('Parse error');
});

test('McpAgentServerCommand_handle_Ugly_rejects_tool_calls_after_quota_is_exhausted', function (): void {
    config()->set('mcp.quota_limit', 1);

    $quotaService = $this->app->make(McpQuotaService::class);
    $quotaService->setQuota('workspace-ugly', 1);
    $quotaService->consume('workspace-ugly');

    $result = mcpAgentServerRun(json_encode([
        'jsonrpc' => '2.0',
        'method' => 'tools/call',
        'params' => [
            'name' => 'echo_tool',
            'arguments' => [
                'workspace_id' => 'workspace-ugly',
                'value' => 'blocked',
            ],
        ],
        'id' => 8,
    ]).PHP_EOL);

    expect($result['exitCode'])->toBe(0)
        ->and($result['responses'])->toHaveCount(1)
        ->and($result['responses'][0]['error']['code'])->toBe(-32001)
        ->and($result['responses'][0]['error']['message'])->toBe('MCP quota exceeded.');
});

test('McpAgentServerCommand_initialize_Good_advertises_the_resources_capability', function (): void {
    // A client that is not told the server has resources never calls
    // resources/list, so an unadvertised surface is undiscoverable.
    $result = mcpAgentServerRun(json_encode([
        'jsonrpc' => '2.0',
        'id' => 1,
        'method' => 'initialize',
    ]).PHP_EOL);

    $capabilities = $result['responses'][0]['result']['capabilities'];

    expect($capabilities)->toHaveKey('resources');
    expect($capabilities['resources']['list'])->toBeTrue();
    expect($capabilities['resources']['read'])->toBeTrue();
    expect($capabilities['tools']['call'])->toBeTrue();
});

test('McpAgentServerCommand_resourcesList_Good_returns_the_agent_resource_surface', function (): void {
    AgentPlan::factory()->create(['slug' => 'listed-plan', 'title' => 'Listed Plan']);

    $result = mcpAgentServerRun(json_encode([
        'jsonrpc' => '2.0',
        'id' => 2,
        'method' => 'resources/list',
    ]).PHP_EOL);

    $uris = array_column($result['responses'][0]['result']['resources'], 'uri');

    // Previously this answered a hardcoded empty array.
    expect($uris)->toContain('plans://all');
    expect($uris)->toContain('plans://listed-plan');
});

test('McpAgentServerCommand_resourcesRead_Good_returns_plan_contents', function (): void {
    AgentPlan::factory()->create(['slug' => 'readable-plan', 'title' => 'Readable Plan']);

    $result = mcpAgentServerRun(json_encode([
        'jsonrpc' => '2.0',
        'id' => 3,
        'method' => 'resources/read',
        'params' => ['uri' => 'plans://all'],
    ]).PHP_EOL);

    $contents = $result['responses'][0]['result']['contents'][0];

    expect($contents['uri'])->toBe('plans://all');
    expect($contents['mimeType'])->toBe('text/markdown');
    expect($contents['text'])->toContain('readable-plan');
});

test('McpAgentServerCommand_resourcesRead_Bad_errors_for_an_unknown_uri', function (): void {
    $result = mcpAgentServerRun(json_encode([
        'jsonrpc' => '2.0',
        'id' => 4,
        'method' => 'resources/read',
        'params' => ['uri' => 'plans://does-not-exist'],
    ]).PHP_EOL);

    // An error, not a body reading "Plan not found" — a client can tell the
    // difference between a missing plan and a plan whose text says that.
    expect($result['responses'][0])->toHaveKey('error');
    expect($result['responses'][0]['error']['code'])->toBe(-32602);
});
