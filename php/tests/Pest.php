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
// This binding covers Feature wholesale, so a Pest file underneath it can never
// declare its own uses(SomeOtherTestCase::class) — Pest raises
// TestCaseAlreadyInUse for the second binding on a file, and does so by class
// name, so a subclass of this TestCase is rejected just the same. Tests needing
// a richer base (the Livewire ones) are written as PHPUnit classes extending it
// instead; class-style tests are not matched by uses()->in() at all.
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
