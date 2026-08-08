<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Blade;
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

    protected function getEnvironmentSetUp($app): void
    {
        $brainDatabase = [
            'driver' => 'sqlite',
            'database' => ':memory:',
            'prefix' => '',
            'foreign_key_constraints' => true,
        ];

        $app['config']->set('mcp.brain.database', $brainDatabase);
        $app['config']->set('database.connections.brain', $brainDatabase);
    }

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

        $this->prepareAgenticLivewireHarness();
    }

    /**
     * Act as the Hades user (admin with full access).
     */
    protected function actingAsHades(): static
    {
        return $this->actingAs($this->hadesUser);
    }

    /**
     * Name a Livewire component class from the module under test.
     *
     * Example:
     *   $component = $this->livewireComponent('FleetOverview');
     */
    protected function livewireComponent(string $component): string
    {
        return "Core\\Mod\\Agentic\\Livewire\\{$component}";
    }

    /**
     * Assert component and Blade wiring without repeating file bootstrap code.
     *
     * Example:
     *   $this->assertFluxComponentWiring('BrainExplorer', 'brain-explorer', [...], [...]);
     *
     * @param  array<int, string>  $componentNeedles
     * @param  array<int, string>  $bladeNeedles
     */
    protected function assertFluxComponentWiring(
        string $component,
        string $bladeName,
        array $componentNeedles,
        array $bladeNeedles,
    ): void {
        $phpRoot = dirname(__DIR__, 3);
        $componentSource = file_get_contents($phpRoot."/Livewire/{$component}.php");
        $bladeSource = file_get_contents(
            $phpRoot."/resources/views/livewire/agentic/{$bladeName}.blade.php",
        );

        expect($componentSource)->toBeString();
        expect($bladeSource)->toBeString();

        foreach ($componentNeedles as $needle) {
            expect($componentSource)->toContain($needle);
        }

        foreach ($bladeNeedles as $needle) {
            expect($bladeSource)->toContain($needle);
        }
    }

    /**
     * Register minimal hub and Flux view stubs for component rendering tests.
     *
     * Example:
     *   $this->prepareAgenticLivewireHarness();
     */
    protected function prepareAgenticLivewireHarness(): void
    {
        $base = sys_get_temp_dir().'/agentic-livewire-stubs';
        $componentPath = $base.'/components';
        $hubPath = $base.'/hub/admin/layouts';

        if (! is_dir($componentPath)) {
            mkdir($componentPath, 0777, true);
        }

        if (! is_dir($hubPath)) {
            mkdir($hubPath, 0777, true);
        }

        file_put_contents($hubPath.'/app.blade.php', '{{ $slot }}');

        $stubs = [
            'badge.blade.php' => '<span {{ $attributes }}>{{ $slot }}</span>',
            'button.blade.php' => '<button {{ $attributes }}>{{ $slot }}</button>',
            'card.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'heading.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'input.blade.php' => '<input {{ $attributes }} />',
            'select.blade.php' => '<select {{ $attributes }}>{{ $slot }}</select>',
            'text.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'textarea.blade.php' => '<textarea {{ $attributes }}>{{ $slot }}</textarea>',
        ];

        foreach ($stubs as $file => $contents) {
            file_put_contents($componentPath.'/'.$file, $contents);
        }

        Blade::anonymousComponentPath($componentPath, 'flux');
        app('view')->addNamespace('hub', $base.'/hub');
    }
}
