<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin;

use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\Button;
use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\Checkbox;
use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\FormGroup;
use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\Input;
use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\Select;
use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\Textarea;
use Core\Mod\Agentic\Mod\Admin\Forms\View\Components\Toggle;
use Core\Mod\Agentic\Mod\Admin\Menu\AdminMenuRegistry;
use Core\Mod\Agentic\Mod\Admin\Search\Providers\AdminPageSearchProvider;
use Core\Mod\Agentic\Mod\Admin\Search\SearchProviderRegistry;
use Core\ModuleRegistry;
use Illuminate\Support\Facades\Blade;
use Illuminate\Support\ServiceProvider;

class Boot extends ServiceProvider
{
    public function register(): void
    {
        if (class_exists(ModuleRegistry::class) && app()->bound(ModuleRegistry::class)) {
            app(ModuleRegistry::class)->addPaths([
                __DIR__.'/../../Website',
            ]);
        }

        $this->app->singleton(SearchProviderRegistry::class);
        $this->app->singleton(AdminMenuRegistry::class);
    }

    public function boot(): void
    {
        $this->loadViewsFrom(__DIR__.'/resources/views', 'core-forms');
        $this->loadTranslationsFrom(__DIR__.'/Lang', 'hub');

        $this->registerFormComponents();
        $this->registerSearchProviders();
    }

    protected function registerFormComponents(): void
    {
        Blade::component('core-forms.input', Input::class);
        Blade::component('core-forms.textarea', Textarea::class);
        Blade::component('core-forms.select', Select::class);
        Blade::component('core-forms.checkbox', Checkbox::class);
        Blade::component('core-forms.button', Button::class);
        Blade::component('core-forms.toggle', Toggle::class);
        Blade::component('core-forms.form-group', FormGroup::class);
    }

    protected function registerSearchProviders(): void
    {
        $this->app->make(SearchProviderRegistry::class)->register(
            $this->app->make(AdminPageSearchProvider::class)
        );
    }
}
