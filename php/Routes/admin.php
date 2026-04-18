<?php

use Illuminate\Support\Facades\Route;
use Mod\Developer\Middleware\RequireHades;

/*
|--------------------------------------------------------------------------
| Agent Operations Routes
|--------------------------------------------------------------------------
*/

Route::prefix('hub')->name('hub.')->group(function () {

    // Agent Operations (Hades only) - protected by middleware
    Route::prefix('agents')
        ->name('agents.')
        ->middleware(['auth', RequireHades::class])
        ->group(function () {
            // Phase 1: Plan Dashboard
            Route::get('/', \Core\Mod\Agentic\View\Modal\Admin\Dashboard::class)->name('index');
            Route::get('/plans', \Core\Mod\Agentic\View\Modal\Admin\Plans::class)->name('plans');
            Route::get('/plans/{slug}', \Core\Mod\Agentic\View\Modal\Admin\PlanDetail::class)->name('plans.show');
            // Phase 2: Session Monitor
            Route::get('/sessions', \Core\Mod\Agentic\View\Modal\Admin\Sessions::class)->name('sessions');
            Route::get('/sessions/{id}', \Core\Mod\Agentic\View\Modal\Admin\SessionDetail::class)->name('sessions.show');
            // Phase 3: Tool Analytics
            Route::get('/tools', \Core\Mod\Agentic\View\Modal\Admin\ToolAnalytics::class)->name('tools');
            Route::get('/tools/calls', \Core\Mod\Agentic\View\Modal\Admin\ToolCalls::class)->name('tools.calls');
            // Phase 4: API Key Management
            Route::get('/api-keys', \Core\Mod\Agentic\View\Modal\Admin\ApiKeys::class)->name('api-keys');
            // Phase 5: Plan Templates
            Route::get('/templates', \Core\Mod\Agentic\View\Modal\Admin\Templates::class)->name('templates');
        });

});
