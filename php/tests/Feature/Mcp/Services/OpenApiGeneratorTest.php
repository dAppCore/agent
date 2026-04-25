<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/OpenApiGenerator.php');

use Core\Mcp\Services\OpenApiGenerator;
use Illuminate\Support\Facades\File;

beforeEach(function (): void {
    File::ensureDirectoryExists(resource_path('mcp/servers'));
});

test('OpenApiGenerator_generate_Good_builds_an_openapi_3_0_3_document_from_registry_files', function (): void {
    File::put(resource_path('mcp/registry.yaml'), "servers:\n  - id: host-hub\n");
    File::put(resource_path('mcp/servers/host-hub.yaml'), "id: host-hub\nname: Host Hub\ntagline: Primary workspace server\n");

    $generator = new OpenApiGenerator;
    $document = $generator->generate();

    expect($document['openapi'])->toBe('3.0.3')
        ->and($document['tags'][2]['name'])->toBe('Host Hub')
        ->and($document['paths'])->toHaveKey('/tools/call');
});

test('OpenApiGenerator_generate_Bad_falls_back_to_registry_ids_when_server_yaml_is_missing', function (): void {
    File::put(resource_path('mcp/registry.yaml'), "servers:\n  - id: marketing\n");
    File::delete(resource_path('mcp/servers/marketing.yaml'));

    $generator = new OpenApiGenerator;
    $document = $generator->generate();

    expect($document['tags'][2]['name'])->toBe('marketing');
});

test('OpenApiGenerator_toJson_Ugly_and_toYaml_keep_the_document_at_openapi_3_0_3_not_3_1', function (): void {
    File::put(resource_path('mcp/registry.yaml'), "servers: []\n");

    $generator = new OpenApiGenerator;

    expect($generator->toJson())->toContain('"openapi": "3.0.3"')
        ->and($generator->toYaml())->toContain("openapi: 3.0.3\n")
        ->and($generator->toYaml())->not->toContain('3.1');
});
