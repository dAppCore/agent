<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpDefineLaravelMcpStubs();
mcpRequire('Mcp/Resources/DatabaseSchema.php');

use Core\Mcp\Resources\DatabaseSchema;
use Laravel\Mcp\Request;

test('DatabaseSchema_handle_Good_serialises_each_table_description_as_pretty_json', function (): void {
    $resource = new class extends DatabaseSchema
    {
        protected function tables(): array
        {
            return ['users'];
        }

        protected function describeTable(string $tableName): array
        {
            return [['Field' => 'id', 'Type' => 'bigint']];
        }
    };

    $payload = json_decode($resource->handle(new Request)->content, true, 512, JSON_THROW_ON_ERROR);

    expect($payload['users'][0]['Field'])->toBe('id');
});

test('DatabaseSchema_handle_Bad_returns_an_empty_json_object_when_no_tables_are_available', function (): void {
    $resource = new class extends DatabaseSchema
    {
        protected function tables(): array
        {
            return [];
        }
    };

    expect($resource->handle(new Request)->content)->toBe('[]');
});

test('DatabaseSchema_handle_Ugly_aggregates_multiple_tables_in_a_single_response', function (): void {
    $resource = new class extends DatabaseSchema
    {
        protected function tables(): array
        {
            return ['users', 'posts'];
        }

        protected function describeTable(string $tableName): array
        {
            return [['table' => $tableName]];
        }
    };

    $payload = json_decode($resource->handle(new Request)->content, true, 512, JSON_THROW_ON_ERROR);

    expect(array_keys($payload))->toBe(['users', 'posts']);
});
