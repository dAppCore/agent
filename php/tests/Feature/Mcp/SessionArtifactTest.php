<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Session\SessionArtifact;
use Core\Mod\Agentic\Models\AgentSession;

it('records artifact descriptions as metadata without throwing a type error', function () {
    $workspace = createWorkspace();
    $session = AgentSession::start(null, AgentSession::AGENT_OPUS, $workspace);
    $tool = new SessionArtifact;

    $payload = [
        'session_id' => $session->session_id,
        'path' => 'docs/session-artifact.md',
        'action' => 'modified',
        'description' => 'some narrative text',
        'metadata' => null,
    ];

    $result = null;

    expect(function () use ($tool, $payload, $session, &$result): void {
        $result = $tool->handle($payload, ['session_id' => $session->session_id]);
    })->not->toThrow(TypeError::class);

    expect($result)->toBeArray()
        ->and($result['success'])->toBeTrue()
        ->and($result['artifact'])->toBe('docs/session-artifact.md');

    $artifacts = $session->fresh()->artifacts;

    expect($artifacts)->toHaveCount(1)
        ->and($artifacts[0]['path'])->toBe('docs/session-artifact.md')
        ->and($artifacts[0]['action'])->toBe('modified')
        ->and($artifacts[0]['metadata'])->toBe(['description' => 'some narrative text']);
});
