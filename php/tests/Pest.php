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

uses(TestCase::class)->in('Feature', 'Unit', 'UseCase');

/*
|--------------------------------------------------------------------------
| Database Refresh
|--------------------------------------------------------------------------
|
| Apply RefreshDatabase to Feature tests that need a clean database state.
| Unit tests typically don't require database access.
|
*/

uses(RefreshDatabase::class)->in('Feature');

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
