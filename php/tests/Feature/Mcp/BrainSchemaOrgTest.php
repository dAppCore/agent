<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mcp\Services\CircuitBreaker;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRecall;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRemember;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Str;

function passThroughBrainCircuitBreaker($app): void
{
    $breaker = Mockery::mock(CircuitBreaker::class);
    $breaker->shouldReceive('call')
        ->andReturnUsing(function (string $service, Closure $operation, ?Closure $fallback = null): mixed {
            return $operation();
        });

    $app->instance(CircuitBreaker::class, $breaker);
}

test('BrainSchemaOrg_brain_remember_Good_accepts_org_and_forwards_it', function (): void {
    $workspace = createWorkspace();
    $brain = new class extends BrainService
    {
        public array $remembered = [];

        public function remember(array $attributes): BrainMemory
        {
            $this->remembered = $attributes;

            $memory = new BrainMemory;
            $memory->forceFill(array_merge([
                'id' => Str::uuid()->toString(),
                'workspace_id' => $attributes['workspace_id'],
                'agent_id' => $attributes['agent_id'],
                'type' => $attributes['type'],
                'content' => $attributes['content'],
                'tags' => $attributes['tags'] ?? [],
                'org' => $attributes['org'] ?? null,
                'project' => $attributes['project'] ?? null,
                'confidence' => $attributes['confidence'] ?? 0.8,
                'supersedes_id' => $attributes['supersedes_id'] ?? null,
            ], $attributes));
            $memory->exists = true;

            return $memory;
        }
    };

    passThroughBrainCircuitBreaker($this->app);
    $this->app->instance(BrainService::class, $brain);

    $tool = new BrainRemember;
    $result = $tool->handle([
        'content' => 'Shared organisation memory.',
        'type' => 'fact',
        'org' => 'core',
    ], [
        'workspace_id' => $workspace->id,
        'session_id' => 'session-1',
    ]);

    expect($tool->inputSchema()['properties'])->toHaveKey('org')
        ->and($result['success'])->toBeTrue()
        ->and($brain->remembered['org'])->toBe('core')
        ->and($result['memory']['org'])->toBe('core');
});

test('BrainSchemaOrg_brain_recall_Bad_accepts_org_filter_and_forwards_it', function (): void {
    $workspace = createWorkspace();
    $brain = new class extends BrainService
    {
        public array $captured = [];

        public function recall(
            string $query,
            int $topK,
            array $filter,
            int $workspaceId,
            array $keywords = [],
            array $boostKeywords = [],
        ): array {
            $this->captured = [
                'query' => $query,
                'topK' => $topK,
                'filter' => $filter,
                'workspace_id' => $workspaceId,
            ];

            return [
                'memories' => [],
                'scores' => [],
            ];
        }
    };

    passThroughBrainCircuitBreaker($this->app);
    $this->app->instance(BrainService::class, $brain);

    $tool = new BrainRecall;
    $result = $tool->handle([
        'query' => 'org-filtered recall',
        'filter' => [
            'org' => 'core',
        ],
    ], [
        'workspace_id' => $workspace->id,
    ]);

    expect($tool->inputSchema()['properties']['filter']['properties'])->toHaveKey('org')
        ->and($result['success'])->toBeTrue()
        ->and($brain->captured['filter']['org'])->toBe('core')
        ->and($brain->captured['workspace_id'])->toBe($workspace->id);
});

test('BrainSchemaOrg_brain_list_Ugly_accepts_org_filter_without_validation_error', function (): void {
    $workspace = createWorkspace();
    passThroughBrainCircuitBreaker($this->app);
    $matching = BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'Core memory.',
        'confidence' => 0.8,
        'org' => 'core',
        'project' => 'agent',
    ]);
    BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'Other org memory.',
        'confidence' => 0.8,
        'org' => 'other-org',
        'project' => 'agent',
    ]);

    $tool = new BrainList;
    $result = $tool->handle([
        'org' => 'core',
    ], [
        'workspace_id' => $workspace->id,
    ]);

    expect($tool->inputSchema()['properties'])->toHaveKey('org')
        ->and($result['success'])->toBeTrue()
        ->and($result['count'])->toBe(1)
        ->and($result['memories'][0]['id'])->toBe($matching->id)
        ->and($result['memories'][0]['org'])->toBe('core');
});
