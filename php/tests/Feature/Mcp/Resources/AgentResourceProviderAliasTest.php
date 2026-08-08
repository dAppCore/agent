<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Services\AgentResourceRegistry;

/*
|--------------------------------------------------------------------------
| AgentResourceProvider alias
|--------------------------------------------------------------------------
|
| dappcore/mcp's McpApiController resolves its plans:// and sessions:// content
| from whatever is bound under Core\Mcp\Resources\Contracts\AgentResourceProvider.
| This package keeps its own copy of Core\Mcp rather than depending on mcp, so
| it cannot name that interface — it binds the registry under the name as a
| string, and mcp accepts a provider structurally.
|
| Without this alias those two endpoint families answer not-found forever,
| which is what "interim" quietly meant before it was made a decision.
|
*/

const PROVIDER_KEY = 'Core\\Mcp\\Resources\\Contracts\\AgentResourceProvider';

it('binds the registry under the provider key mcp resolves', function (): void {
    expect(app()->bound(PROVIDER_KEY))->toBeTrue()
        ->and(app(PROVIDER_KEY))->toBeInstanceOf(AgentResourceRegistry::class);
});

it('is the same instance as the registry itself', function (): void {
    // An alias, not a second binding: two registries would drift.
    expect(app(PROVIDER_KEY))->toBe(app(AgentResourceRegistry::class));
});

it('satisfies the one method mcp calls on it', function (): void {
    AgentPlan::factory()->create(['slug' => 'aliased-plan', 'title' => 'Aliased Plan']);

    $contents = app(PROVIDER_KEY)->read('plans://all');

    expect($contents)->not->toBeNull()
        ->and($contents['mimeType'])->toBe('text/markdown')
        ->and($contents['text'])->toContain('aliased-plan');
});

it('reports nothing for a uri it does not serve', function (): void {
    // mcp turns null into a clean not-found rather than a body saying so.
    expect(app(PROVIDER_KEY)->read('nonsense://thing'))->toBeNull();
});
