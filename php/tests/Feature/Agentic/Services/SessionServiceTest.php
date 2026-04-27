<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Services\SessionService;
use Illuminate\Support\Facades\Event;

if (! function_exists('loadAgenticPhpClass')) {
    function loadAgenticPhpClass(string $relativePath): void
    {
        $phpRoot = dirname(__DIR__, 4);
        require_once $phpRoot.'/'.$relativePath;
    }
}

beforeEach(function (): void {
    loadAgenticPhpClass('Agentic/Services/SessionService.php');
});

test('SessionService_create_Good_creates_active_sessions_and_emits_sse_frames', function (): void {
    $workspace = createWorkspace();
    $service = new SessionService();
    $captured = [];

    Event::listen(SessionService::SSE_EVENT, function (array $payload) use (&$captured): void {
        $captured[] = $payload;
    });

    $session = $service->create($workspace->id, [
        'agent_type' => 'opus',
        'context_summary' => ['goal' => 'Verify the queue'],
    ]);

    $paused = $service->updateState($session, AgentSession::STATUS_PAUSED);
    $frame = $service->emit($paused, [
        'event' => 'session.checkpoint',
        'data' => ['checkpoint' => 'awaiting-review'],
    ]);

    expect($session->status)->toBe(AgentSession::STATUS_ACTIVE)
        ->and($paused->status)->toBe(AgentSession::STATUS_PAUSED)
        ->and($frame)->toContain('event: session.checkpoint')
        ->and($frame)->toContain('"checkpoint":"awaiting-review"')
        ->and($captured)->toHaveCount(3)
        ->and($captured[0]['event'])->toBe('session.created')
        ->and($captured[1]['event'])->toBe('session.state.updated')
        ->and($captured[2]['event'])->toBe('session.checkpoint');
});

test('SessionService_updateState_Bad_rejects_invalid_terminal_state_transitions', function (): void {
    $workspace = createWorkspace();
    $service = new SessionService();
    $session = $service->create($workspace->id, ['agent_type' => 'sonnet']);

    $completed = $service->updateState($session, 'closed');

    expect($completed->status)->toBe(AgentSession::STATUS_COMPLETED);

    expect(fn () => $service->updateState($completed, AgentSession::STATUS_ACTIVE))
        ->toThrow(InvalidArgumentException::class, 'Invalid session transition [completed -> active]');
});

test('SessionService_updateState_Ugly_allows_handoff_reactivation_by_session_id', function (): void {
    $workspace = createWorkspace();
    $service = new SessionService();
    $session = $service->create($workspace->id, [
        'agent_type' => 'haiku',
        'handoff_notes' => ['summary' => 'Ready for follow-up'],
    ]);

    $handedOff = $service->updateState($session, AgentSession::STATUS_HANDED_OFF);
    $reactivated = $service->updateState($handedOff->session_id, 'resumed');

    expect($handedOff->status)->toBe(AgentSession::STATUS_HANDED_OFF)
        ->and($reactivated->status)->toBe(AgentSession::STATUS_ACTIVE)
        ->and($reactivated->session_id)->toBe($session->session_id)
        ->and($reactivated->last_active_at)->not->toBeNull();
});
