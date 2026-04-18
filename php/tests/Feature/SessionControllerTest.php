<?php

declare(strict_types=1);

use Core\Mod\Agentic\Controllers\Api\SessionController;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;

it('ends a session with handoff notes', function (): void {
    $workspace = Workspace::factory()->create();
    $session = AgentSession::factory()->active()->create([
        'workspace_id' => $workspace->id,
    ]);

    $request = Request::create('/v1/sessions/'.$session->session_id.'/end', 'POST', [
        'status' => 'handed_off',
        'summary' => 'Ready for review',
        'handoff_notes' => [
            'summary' => 'Ready for review',
            'next_steps' => ['Run the verifier'],
            'blockers' => ['Need approval'],
            'context_for_next' => ['repo' => 'core/go-io'],
        ],
    ]);
    $request->attributes->set('workspace', $workspace);

    $response = app(SessionController::class)->end($request, $session->session_id);

    expect($response->getStatusCode())->toBe(200);

    $payload = $response->getData(true);

    expect($payload['data']['session_id'])->toBe($session->session_id)
        ->and($payload['data']['status'])->toBe(AgentSession::STATUS_HANDED_OFF)
        ->and($payload['data']['final_summary'])->toBe('Ready for review')
        ->and($payload['data']['handoff_notes']['summary'])->toBe('Ready for review')
        ->and($payload['data']['handoff_notes']['next_steps'])->toBe(['Run the verifier']);

    $fresh = $session->fresh();

    expect($fresh?->status)->toBe(AgentSession::STATUS_HANDED_OFF)
        ->and($fresh?->final_summary)->toBe('Ready for review')
        ->and($fresh?->handoff_notes['context_for_next'])->toBe(['repo' => 'core/go-io']);
});
