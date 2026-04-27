<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainForget;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRemember;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Queue;

function brainSmokeService(): BrainService
{
    return new class extends BrainService
    {
        public bool $forgetCalled = false;

        public function forget(string $id): void
        {
            $this->forgetCalled = true;

            DB::connection('brain')->transaction(function () use ($id): void {
                BrainMemory::where('id', $id)->delete();
            });
        }
    };
}

/**
 * @return array{workspace: \Core\Tenant\Models\Workspace, session: AgentSession, context: array{workspace_id: int, session_id: string}}
 */
function brainSmokeContext(): array
{
    $workspace = createWorkspace();
    $session = AgentSession::start(null, AgentSession::AGENT_OPUS, $workspace);

    return [
        'workspace' => $workspace,
        'session' => $session,
        'context' => [
            'workspace_id' => $workspace->id,
            'session_id' => $session->session_id,
        ],
    ];
}

test('BrainSmoke_remember_list_forget_Good_exercises_mariadb_only_handlers_end_to_end', function (): void {
    Queue::fake();

    $brain = brainSmokeService();
    $this->app->instance(BrainService::class, $brain);

    [
        'workspace' => $workspace,
        'session' => $session,
        'context' => $context,
    ] = brainSmokeContext();

    $rememberPayload = [
        'content' => 'test memory about X',
        'scope' => 'workspace',
        'type' => 'context',
        'tags' => ['smoketest'],
    ];

    $rememberResult = (new BrainRemember)->handle($rememberPayload, $context);

    expect($rememberResult)->toBeArray()
        ->and($rememberResult['success'])->toBeTrue()
        ->and($rememberResult['memory']['id'])->toBeString()
        ->and($rememberResult['memory']['content'])->toBe($rememberPayload['content'])
        ->and($rememberResult['memory']['type'])->toBe($rememberPayload['type'])
        ->and($rememberResult['memory']['tags'])->toBe($rememberPayload['tags'])
        ->and($rememberResult['memory']['agent_id'])->toBe($session->session_id);

    $memoryId = $rememberResult['memory']['id'];
    $storedMemory = BrainMemory::query()->find($memoryId);

    expect($storedMemory)->not->toBeNull()
        ->and($storedMemory?->workspace_id)->toBe($workspace->id)
        ->and($storedMemory?->content)->toBe($rememberPayload['content'])
        ->and($storedMemory?->type)->toBe($rememberPayload['type'])
        ->and($storedMemory?->tags)->toBe($rememberPayload['tags'])
        ->and($storedMemory?->deleted_at)->toBeNull();

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memoryId);

    $listResult = (new BrainList)->handle([
        'scope' => 'workspace',
    ], $context);

    expect($listResult)->toBeArray()
        ->and($listResult['success'])->toBeTrue()
        ->and($listResult['count'])->toBe(1)
        ->and($listResult['memories'])->toHaveCount(1)
        ->and($listResult['memories'][0]['id'])->toBe($memoryId)
        ->and($listResult['memories'][0]['agent_id'])->toBe($session->session_id)
        ->and($listResult['memories'][0]['content'])->toBe($rememberPayload['content'])
        ->and($listResult['memories'][0]['type'])->toBe($rememberPayload['type'])
        ->and($listResult['memories'][0]['tags'])->toBe($rememberPayload['tags']);

    $forgetResult = (new BrainForget)->handle([
        'id' => $memoryId,
    ], $context);

    expect($forgetResult)->toBeArray()
        ->and($forgetResult['success'])->toBeTrue()
        ->and($forgetResult['forgotten'])->toBe($memoryId)
        ->and($forgetResult['type'])->toBe($rememberPayload['type'])
        ->and($brain->forgetCalled)->toBeTrue();

    $deletedMemory = BrainMemory::withTrashed()->find($memoryId);

    expect(BrainMemory::query()->find($memoryId))->toBeNull()
        ->and($deletedMemory)->not->toBeNull()
        ->and($deletedMemory?->trashed())->toBeTrue();
});

test('BrainSmoke_forget_Bad_reports_missing_memory_without_a_type_error', function (): void {
    $brain = brainSmokeService();
    $this->app->instance(BrainService::class, $brain);

    ['context' => $context] = brainSmokeContext();

    $missingId = '00000000-0000-4000-8000-000000000096';
    $tool = new BrainForget;
    $result = null;
    $reportedError = null;

    try {
        $result = $tool->handle(['id' => $missingId], $context);
    } catch (\InvalidArgumentException $exception) {
        $reportedError = $exception->getMessage();
    }

    expect($reportedError ?? $result['error'] ?? null)->toBe("Memory '{$missingId}' not found in this workspace")
        ->and($result['success'] ?? false)->toBeFalse()
        ->and($brain->forgetCalled)->toBeFalse();
});
