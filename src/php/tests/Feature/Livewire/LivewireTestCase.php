<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\Fixtures\HadesUser;
use Tests\TestCase;

/**
 * Base test case for Livewire component tests.
 *
 * Registers stub view namespaces so components can render during tests
 * without requiring the full hub/mcp Blade component library.
 */
abstract class LivewireTestCase extends TestCase
{
    use RefreshDatabase;

    protected HadesUser $hadesUser;

    protected function setUp(): void
    {
        parent::setUp();

        // Register stub view namespaces so Livewire can render components
        // without the full Blade component library from host-uk/core.
        // Stubs live in tests/views/{namespace}/ and use minimal HTML.
        $viewsBase = realpath(__DIR__.'/../../views');

        $this->app['view']->addNamespace('agentic', $viewsBase);
        $this->app['view']->addNamespace('mcp', $viewsBase.'/mcp');

        // Create a Hades-privileged user for component tests
        $this->hadesUser = new HadesUser([
            'id' => 1,
            'name' => 'Hades Test User',
            'email' => 'hades@test.example',
        ]);
    }

    /**
     * Act as the Hades user (admin with full access).
     */
    protected function actingAsHades(): static
    {
        return $this->actingAs($this->hadesUser);
    }
}
