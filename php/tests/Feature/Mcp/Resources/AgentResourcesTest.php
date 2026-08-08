<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Resources\Agent\AllPlansResource;
use Core\Mod\Agentic\Mcp\Resources\Agent\PhaseChecklistResource;
use Core\Mod\Agentic\Mcp\Resources\Agent\PlanDocumentResource;
use Core\Mod\Agentic\Mcp\Resources\Agent\SessionContextResource;
use Core\Mod\Agentic\Mcp\Resources\Agent\StateValueResource;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Services\AgentResourceRegistry;

/*
|--------------------------------------------------------------------------
| MCP agent resources
|--------------------------------------------------------------------------
|
| The plans/phases/sessions/state resource surface ported out of
| dappcore/mcp's McpAgentServerCommand. Before this, the agent server answered
| resources/list with a hardcoded empty array and had no resources/read at all,
| so none of it existed on this side.
|
*/

function agentResourceRegistry(): AgentResourceRegistry
{
    return (new AgentResourceRegistry)->registerMany([
        new AllPlansResource,
        new PlanDocumentResource,
        new PhaseChecklistResource,
        new StateValueResource,
        new SessionContextResource,
    ]);
}

it('routes each URI shape to exactly one resource', function (): void {
    $registry = agentResourceRegistry();

    expect($registry->resolve('plans://all'))->toBeInstanceOf(AllPlansResource::class)
        ->and($registry->resolve('plans://my-plan'))->toBeInstanceOf(PlanDocumentResource::class)
        ->and($registry->resolve('plans://my-plan/phases/2'))->toBeInstanceOf(PhaseChecklistResource::class)
        ->and($registry->resolve('plans://my-plan/state/build_id'))->toBeInstanceOf(StateValueResource::class)
        ->and($registry->resolve('sessions://abc123/context'))->toBeInstanceOf(SessionContextResource::class);
});

it('claims no URI outside the shapes it serves', function (): void {
    $registry = agentResourceRegistry();

    // plans://all must not be swallowed by the {slug} resource, and a shape
    // that is nearly right must not be served with the wrong reader.
    expect($registry->resolve('plans://'))->toBeNull()
        ->and($registry->resolve('plans://my-plan/phases/latest'))->toBeNull()
        ->and($registry->resolve('plans://my-plan/state/'))->toBeNull()
        ->and($registry->resolve('sessions://abc123'))->toBeNull()
        ->and($registry->resolve('nonsense://thing'))->toBeNull();
});

it('reads the all-plans overview', function (): void {
    AgentPlan::factory()->create(['slug' => 'first-plan', 'title' => 'First Plan']);

    $contents = agentResourceRegistry()->read('plans://all');

    expect($contents)->not->toBeNull()
        ->and($contents['uri'])->toBe('plans://all')
        ->and($contents['mimeType'])->toBe('text/markdown')
        ->and($contents['text'])->toContain('# Work Plans')
        ->and($contents['text'])->toContain('first-plan');
});

it('reads one plan document', function (): void {
    AgentPlan::factory()->create(['slug' => 'ship-it', 'title' => 'Ship It']);

    $contents = agentResourceRegistry()->read('plans://ship-it');

    expect($contents)->not->toBeNull()
        ->and($contents['uri'])->toBe('plans://ship-it')
        ->and($contents['text'])->toBeString();
});

it('advertises every non-archived plan in the listing', function (): void {
    AgentPlan::factory()->create(['slug' => 'listed-plan', 'title' => 'Listed Plan']);

    $uris = array_column(agentResourceRegistry()->entries(), 'uri');

    // The static overview plus a dynamic entry per plan — a client should see
    // the plans that exist, not a template it has to guess slugs for.
    expect($uris)->toContain('plans://all')
        ->and($uris)->toContain('plans://listed-plan');
});

it('returns null rather than a document when the target does not exist', function (): void {
    $registry = agentResourceRegistry();

    // The shape matches, so a resource claims it; the plan does not exist, so
    // the read reports nothing. Previously this returned the string "Plan not
    // found: ..." AS the resource body, which a client cannot tell apart from
    // a real document that happens to say so.
    expect($registry->resolve('plans://ghost'))->not->toBeNull()
        ->and($registry->read('plans://ghost'))->toBeNull()
        ->and($registry->read('plans://ghost/phases/1'))->toBeNull()
        ->and($registry->read('plans://ghost/state/key'))->toBeNull()
        ->and($registry->read('sessions://ghost/context'))->toBeNull();
});
