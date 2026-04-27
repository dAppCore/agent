<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mod/Mcp/Services/AgentSessionService.php');

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Illuminate\Support\Facades\Cache;
use Mod\Mcp\Services\AgentSessionService;

beforeEach(function (): void {
    Cache::flush();
});

test('AgentSessionService_start_Good_creates_and_caches_a_new_active_session', function (): void {
    $workspace = createWorkspace();
    $plan = AgentPlan::factory()->create(['workspace_id' => $workspace->id]);
    $service = new AgentSessionService;

    $session = $service->start('opus', $plan, $workspace->id, ['task' => 'draft']);

    expect($session)->toBeInstanceOf(AgentSession::class)
        ->and($service->exists($session->session_id))->toBeTrue()
        ->and(Cache::get('mcp_session:active:'.$session->session_id)['workspace_id'])->toBe($workspace->id);
});

test('AgentSessionService_end_Bad_returns_null_for_unknown_sessions', function (): void {
    $service = new AgentSessionService;

    expect($service->end('missing-session'))->toBeNull()
        ->and($service->getHandoffContext('missing-session'))->toBeNull();
});

test('AgentSessionService_continueFrom_Ugly_hands_off_the_previous_session_and_starts_a_follow_up', function (): void {
    $workspace = createWorkspace();
    $plan = AgentPlan::factory()->create(['workspace_id' => $workspace->id]);
    $service = new AgentSessionService;
    $first = $service->start('opus', $plan, $workspace->id, ['task' => 'phase-one']);

    $service->prepareHandoff($first->session_id, 'Done', ['next'], ['none'], ['checkpoint' => 'alpha']);
    $second = $service->continueFrom($first->session_id, 'sonnet');

    expect($second)->toBeInstanceOf(AgentSession::class)
        ->and($first->fresh()->status)->toBe(AgentSession::STATUS_HANDED_OFF)
        ->and($second->context_summary['continued_from'])->toBe($first->session_id)
        ->and($second->context_summary['previous_agent'])->toBe('opus');
});
