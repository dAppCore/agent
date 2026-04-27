<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mod\Admin;

use Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase;
use Illuminate\Support\Facades\Route;
use Livewire\Livewire;

abstract class AdminTestCase extends LivewireTestCase
{
    protected function getPackageProviders($app): array
    {
        return array_merge(parent::getPackageProviders($app), [
            \Core\Mod\Agentic\Mod\Admin\Boot::class,
        ]);
    }

    protected function setUp(): void
    {
        parent::setUp();

        $basePath = dirname(__DIR__, 4);
        $this->app['view']->addNamespace('hub', $basePath.'/Website/Hub/View/Blade');

        if (! Route::has('hub.dashboard')) {
            Route::middleware('web')
                ->prefix('hub')
                ->name('hub.')
                ->group($basePath.'/Website/Hub/Routes/admin.php');
        }

        Livewire::component('hub.admin.workspace-switcher', \Core\Mod\Agentic\Website\Hub\View\Modal\Admin\WorkspaceSwitcher::class);
        Livewire::component('hub.admin.global-search', \Core\Mod\Agentic\Website\Hub\View\Modal\Admin\GlobalSearch::class);
    }
}
