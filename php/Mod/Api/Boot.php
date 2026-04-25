<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api;

use Core\Events\ApiRoutesRegistering;
use Core\Mod\Agentic\Mod\Api\Documentation\Middleware\ProtectDocumentation;
use Core\Mod\Agentic\Mod\Api\RateLimit\RateLimitService;
use Illuminate\Contracts\Cache\Repository as CacheRepository;
use Illuminate\Support\ServiceProvider;

class Boot extends ServiceProvider
{
    /**
     * Events this module listens to for lazy loading.
     *
     * @var array<class-string, string>
     */
    public static array $listens = [
        ApiRoutesRegistering::class => 'onApiRoutes',
    ];

    public function register(): void
    {
        $this->app->singleton(RateLimitService::class, function ($app): RateLimitService {
            return new RateLimitService($app->make(CacheRepository::class));
        });

        $this->app->singleton(Services\WebhookSignature::class);
        $this->app->singleton(Services\WebhookService::class);
    }

    public function onApiRoutes(ApiRoutesRegistering $event): void
    {
        $event->middleware('api.auth', Middleware\AuthenticateApiKey::class);
        $event->middleware('auth.api', Middleware\AuthenticateApiKey::class);
        $event->middleware('api.docs.protect', ProtectDocumentation::class);

        if (file_exists(__DIR__.'/Routes/api.php')) {
            $event->routes(fn () => require __DIR__.'/Routes/api.php');
        }
    }
}
