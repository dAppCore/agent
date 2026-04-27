<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpDefineLaravelMcpStubs();
mcpRequire('Mcp/Resources/AppConfig.php');

use Core\Mcp\Resources\AppConfig;
use Laravel\Mcp\Request;

test('AppConfig_handle_Good_returns_application_config_as_pretty_printed_json', function (): void {
    config()->set('app.name', 'Host Hub');
    config()->set('app.env', 'testing');
    config()->set('app.debug', true);
    config()->set('app.url', 'https://host.test');

    $response = (new AppConfig)->handle(new Request);
    $payload = json_decode($response->content, true, 512, JSON_THROW_ON_ERROR);

    expect($payload['name'])->toBe('Host Hub')
        ->and($payload['url'])->toBe('https://host.test');
});

test('AppConfig_handle_Bad_keeps_missing_values_as_nulls_instead_of_throwing', function (): void {
    config()->set('app.name', null);
    config()->set('app.url', null);

    $payload = json_decode((new AppConfig)->handle(new Request)->content, true, 512, JSON_THROW_ON_ERROR);

    expect($payload['name'])->toBeNull()
        ->and($payload['url'])->toBeNull();
});

test('AppConfig_handle_Ugly_preserves_json_encoding_for_slash_containing_urls', function (): void {
    config()->set('app.url', 'https://host.test/api/v1');

    $content = (new AppConfig)->handle(new Request)->content;

    expect($content)->toContain('https://host.test/api/v1');
});
