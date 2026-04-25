<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\Support;

use Illuminate\Support\Facades\Route;

final class HubRouteNames
{
    public static function prefix(?string $host = null): string
    {
        $host ??= request()->getHost();

        if ($host === null || $host === '') {
            return 'hub.';
        }

        if ((bool) preg_match('/^core\.(test|localhost)$/', $host)) {
            return 'hub.';
        }

        return str_replace('.', '_', $host).'.';
    }

    public static function name(string $suffix, ?string $host = null): string
    {
        return self::prefix($host).$suffix;
    }

    /**
     * @param  array<string, mixed>  $parameters
     */
    public static function url(string $suffix, array $parameters = [], ?string $fallback = null, ?string $host = null): string
    {
        $name = self::name($suffix, $host);

        if (Route::has($name)) {
            return route($name, $parameters);
        }

        return $fallback ?? '#';
    }
}
