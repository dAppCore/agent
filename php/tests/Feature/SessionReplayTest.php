<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Session\SessionReplay;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;

uses(RefreshDatabase::class);

beforeEach(function () {
    $this->workspace = Workspace::factory()->create();
});

it('returns replay context for a stored session', function () {
    $session = AgentSession::create([
        'workspace_id' => $this->workspace->id,
        'session_id' => 'ses_test_replay',
        'agent_type' => 'opus',
        'status' => AgentSession::STATUS_FAILED,
        'context_summary' => ['goal' => 'Investigate replay'],
        'work_log' => [
            [
                'message' => 'Reached parser step',
                'type' => 'checkpoint',
                'data' => ['step' => 2],
                'timestamp' => now()->subMinutes(20)->toIso8601String(),
            ],
            [
                'message' => 'Chose retry path',
                'type' => 'decision',
                'data' => ['path' => 'retry'],
                'timestamp' => now()->subMinutes(10)->toIso8601String(),
            ],
            [
                'message' => 'Vector store timeout',
                'type' => 'error',
                'data' => ['service' => 'qdrant'],
                'timestamp' => now()->subMinutes(5)->toIso8601String(),
            ],
        ],
        'artifacts' => [
            [
                'path' => 'README.md',
                'action' => 'modified',
                'metadata' => ['bytes' => 128],
                'timestamp' => now()->subMinutes(8)->toIso8601String(),
            ],
        ],
        'started_at' => now()->subHour(),
        'last_active_at' => now()->subMinutes(5),
        'ended_at' => now()->subMinutes(1),
    ]);

    $tool = new SessionReplay;
    $result = $tool->handle(['session_id' => $session->session_id]);

    expect($result)->toBeArray()
        ->and($result['success'])->toBeTrue()
        ->and($result)->toHaveKey('replay_context');

    $context = $result['replay_context'];

    expect($context['session_id'])->toBe($session->session_id)
        ->and($context['last_checkpoint']['message'])->toBe('Reached parser step')
        ->and($context['decisions'])->toHaveCount(1)
        ->and($context['errors'])->toHaveCount(1)
        ->and($context['progress_summary']['checkpoint_count'])->toBe(1)
        ->and($context['artifacts_by_action']['modified'])->toHaveCount(1);
});

it('declares read scope', function () {
    $tool = new SessionReplay;

    expect($tool->requiredScopes())->toBe(['read']);
});

it('returns an error for an unknown session', function () {
    $tool = new SessionReplay;
    $result = $tool->handle(['session_id' => 'missing-session']);

    expect($result)->toBeArray()
        ->and($result['error'])->toBe('Session not found: missing-session');
});
