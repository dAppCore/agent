<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpDefineLaravelMcpStubs();
mcpRequire('Mcp/Resources/ContentResource.php');

use Core\Mcp\Resources\ContentResource;
use Illuminate\Support\Collection;
use Laravel\Mcp\Request;

test('ContentResource_handle_Good_renders_frontmatter_excerpt_and_markdown_body', function (): void {
    $workspace = (object) ['id' => 1, 'slug' => 'host'];
    $item = (object) [
        'title' => 'Launch Post',
        'slug' => 'launch-post',
        'type' => 'post',
        'status' => 'publish',
        'author' => (object) ['name' => 'Virgil'],
        'categories' => [(object) ['name' => 'News']],
        'tags' => [(object) ['name' => 'Launch']],
        'excerpt' => 'Big launch.',
        'content_markdown' => '# Hello',
        'created_at' => now(),
        'updated_at' => now(),
    ];

    $resource = new class($workspace, $item) extends ContentResource
    {
        public function __construct(private object $workspaceFixture, private object $itemFixture)
        {
        }

        protected function resolveWorkspace(string $identifier): ?object
        {
            return $this->workspaceFixture;
        }

        protected function resolveContentItem(object $workspace, string $identifier): ?object
        {
            return $this->itemFixture;
        }
    };

    $content = $resource->handle(new Request(['uri' => 'content://host/launch-post']))->content;

    expect($content)->toContain("title: Launch Post")
        ->and($content)->toContain('> Big launch.')
        ->and($content)->toContain('# Hello');
});

test('ContentResource_handle_Bad_rejects_invalid_content_uris', function (): void {
    $resource = new class extends ContentResource {};

    expect($resource->handle(new Request(['uri' => 'invalid://x']))->content)
        ->toBe('Invalid URI format. Expected: content://{workspace}/{slug}');
});

test('ContentResource_list_Ugly_can_be_overridden_to_surface_resource_lists_without_the_database', function (): void {
    $resource = new class extends ContentResource
    {
        protected function listResources(): array
        {
            return [[
                'uri' => 'content://host/launch-post',
                'name' => 'Launch Post',
                'description' => 'Post: Launch Post',
                'mimeType' => 'text/markdown',
            ]];
        }
    };

    expect(mcpInvokeProtected($resource, 'listResources'))->toBe([[
        'uri' => 'content://host/launch-post',
        'name' => 'Launch Post',
        'description' => 'Post: Launch Post',
        'mimeType' => 'text/markdown',
    ]]);
});
