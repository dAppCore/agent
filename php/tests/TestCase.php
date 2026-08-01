<?php

declare(strict_types=1);

namespace Tests;

use Core\Mod\Agentic\Boot as AgenticBoot;
use Core\Tenant\Boot as TenantBoot;
use Livewire\LivewireServiceProvider;
use Orchestra\Testbench\TestCase as BaseTestCase;
use Spatie\Activitylog\ActivitylogServiceProvider;

abstract class TestCase extends BaseTestCase
{
    /**
     * Core\Tenant\Boot is registered alongside Agentic because almost every
     * model in this package is workspace-scoped: without the tenant provider
     * there is no workspaces table, so anything that touches one dies on a
     * missing table rather than on the behaviour under test.
     */
    protected function getPackageProviders($app): array
    {
        return [
            // The models use Spatie's LogsActivity trait, whose causer resolver
            // reads activitylog config at construction — without the provider
            // that config is absent and every save fails inside the trait.
            ActivitylogServiceProvider::class,
            // Livewire is a require-dev dependency and this package ships
            // Livewire components; Testbench does not auto-discover it, so
            // without the provider the `livewire` container binding is absent
            // and every component render dies on "Target class [livewire]
            // does not exist" rather than on the behaviour under test.
            LivewireServiceProvider::class,
            TenantBoot::class,
            AgenticBoot::class,
        ];
    }

    /**
     * Activity logging is off under test.
     *
     * Spatie ships its activity_log schema as .stub files for the host app to
     * publish, so the table cannot exist in a package test run; with logging
     * enabled every model write fails on the missing table. What the trait
     * records is host-application audit history, not behaviour this package
     * asserts, so it is disabled rather than given a synthetic table.
     */
    protected function defineEnvironment($app): void
    {
        $app['config']->set('activitylog.enabled', false);

        // Testbench boots a bare app with no APP_KEY, and the encrypter is
        // reached through the session middleware every rendered view runs
        // through — so without a key the failure surfaced as a ViewException
        // ("No application encryption key has been specified") on any test
        // that rendered anything. A fixed key, not a random one: it keeps runs
        // reproducible and nothing here encrypts data that outlives the test.
        // Exactly 32 bytes — AES-256-CBC rejects any other length.
        $app['config']->set('app.key', 'base64:'.base64_encode('core-agent-testing-key-32bytes!!'));
    }

    /**
     * Run both packages' migrations against the in-memory database.
     *
     * Both providers publish their migrations for a host application to run
     * rather than loading them themselves, so under Testbench there is no
     * schema at all unless the directories are named here.
     */
    protected function defineDatabaseMigrations(): void
    {
        $this->loadMigrationsFrom(dirname(__DIR__, 2).'/vendor/dappcore/php-tenant/Migrations');
        $this->loadMigrationsFrom(dirname(__DIR__).'/Migrations');
    }
}
