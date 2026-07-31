<?php

declare(strict_types=1);

/*
|--------------------------------------------------------------------------
| Pest Configuration
|--------------------------------------------------------------------------
|
| Configure Pest testing framework for the core-agentic package.
| This file binds test traits to test cases and provides helper functions.
|
*/

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Tenant\Models\Workspace;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/*
|--------------------------------------------------------------------------
| Test Case
|--------------------------------------------------------------------------
|
| The closure passed to the "uses()" method binds an abstract test case
| to all Feature and Unit tests. The TestCase class provides a bridge
| between Laravel's testing utilities and Pest's expressive syntax.
|
*/

// __DIR__, not bare names: Pest resolves a bare 'Feature' against its default
// test path (./tests), but this suite lives at php/tests. Unanchored, nothing
// matched, so no TestCase was bound, no Testbench app booted, and every test
// died on a null Eloquent connection resolver.
//
// Feature/Agentic/Livewire is excluded in phpunit.xml rather than bound here:
// those three files each declare uses(LivewireTestCase::class) at file level,
// and although LivewireTestCase extends this same TestCase, Pest compares the
// bound class by name rather than by inheritance and rejects the overlap either
// way round. Untangling that is a change to the Livewire suite, not to this
// binding, so it is left alone and called out in phpunit.xml.
uses(TestCase::class)->in(__DIR__.'/Feature', __DIR__.'/Unit', __DIR__.'/UseCase');

/*
|--------------------------------------------------------------------------
| Database Refresh
|--------------------------------------------------------------------------
|
| Apply RefreshDatabase to Feature tests that need a clean database state.
| Unit tests typically don't require database access.
|
*/

uses(RefreshDatabase::class)->in(__DIR__.'/Feature');

/*
|--------------------------------------------------------------------------
| Helper Functions
|--------------------------------------------------------------------------
|
| Custom helper functions for agent-related tests.
|
*/

/**
 * Create a workspace for testing.
 */
function createWorkspace(array $attributes = []): Workspace
{
    return Workspace::factory()->create($attributes);
}

/**
 * Create an API key for testing.
 */
function createApiKey(
    Workspace|int|null $workspace = null,
    string $name = 'Test Key',
    array $permissions = [],
    int $rateLimit = 100
): AgentApiKey {
    $workspace ??= createWorkspace();

    return AgentApiKey::generate($workspace, $name, $permissions, $rateLimit);
}
