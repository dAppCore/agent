<?php

// SPDX-License-Identifier: EUPL-1.2

use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\AccountUsage;
use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\Dashboard;
use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\Platform;
use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\PlatformUser;
use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\SiteSettings;
use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\Sites;
use Illuminate\Support\Facades\Route;

Route::get('/', Dashboard::class)->name('dashboard');
Route::redirect('/dashboard', '/hub')->name('dashboard.redirect');
Route::get('/workspaces', Sites::class)->name('sites');
Route::redirect('/sites', '/hub/workspaces');
Route::get('/workspaces/{workspace}/{tab?}', SiteSettings::class)
    ->where('tab', 'services|general|deployment|environment|ssl|backups|danger')
    ->name('sites.settings');
Route::get('/account/usage', AccountUsage::class)->name('account.usage');
Route::redirect('/usage', '/hub/account/usage');
Route::redirect('/boosts', '/hub/account/usage?tab=boosts');
Route::redirect('/ai-services', '/hub/account/usage?tab=ai');
Route::get('/platform', Platform::class)->name('platform');
Route::get('/platform/user/{id}', PlatformUser::class)->where('id', '[0-9]+')->name('platform.user');
