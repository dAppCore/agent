<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Search\Providers;

use Core\Mod\Agentic\Mod\Admin\Search\Contracts\SearchProvider;
use Core\Mod\Agentic\Mod\Admin\Search\SearchProviderRegistry;
use Core\Mod\Agentic\Mod\Admin\Search\SearchResult;
use Illuminate\Support\Facades\Route;

class AdminPageSearchProvider implements SearchProvider
{
    public function __construct(
        protected SearchProviderRegistry $registry
    ) {}

    public function name(): string
    {
        return 'Pages';
    }

    public function icon(): string
    {
        return 'rectangle-stack';
    }

    public function priority(): int
    {
        return 10;
    }

    public function available(?object $user = null, ?object $workspace = null): bool
    {
        return $user !== null;
    }

    /**
     * @return array<int, SearchResult>
     */
    public function search(string $query): array
    {
        $pages = array_filter($this->pagesForCurrentUser(), function (array $page) use ($query): bool {
            return $this->registry->fuzzyMatch($query, $page['title'])
                || $this->registry->fuzzyMatch($query, $page['description']);
        });

        usort($pages, fn (array $left, array $right): int => $this->registry->relevanceScore($query, $right['title']) <=> $this->registry->relevanceScore($query, $left['title']));

        return array_map(static fn (array $page): SearchResult => SearchResult::fromArray($page), $pages);
    }

    /**
     * @return array<int, array<string, string>>
     */
    protected function pagesForCurrentUser(): array
    {
        $pages = [
            [
                'id' => 'dashboard',
                'type' => 'admin_page',
                'title' => 'Dashboard',
                'description' => 'Hub landing page',
                'url' => $this->safeRoute('hub.dashboard', '/hub'),
                'icon' => 'house',
            ],
            [
                'id' => 'workspaces',
                'type' => 'admin_page',
                'title' => 'Workspaces',
                'description' => 'Workspace and tenant registry',
                'url' => $this->safeRoute('hub.sites', '/hub/workspaces'),
                'icon' => 'folders',
            ],
            [
                'id' => 'usage',
                'type' => 'admin_page',
                'title' => 'Account Usage',
                'description' => 'Storage, boosts and AI service usage',
                'url' => $this->safeRoute('hub.account.usage', '/hub/account/usage'),
                'icon' => 'chart-pie',
            ],
        ];

        if (auth()->user()?->isHades()) {
            $pages[] = [
                'id' => 'platform',
                'type' => 'admin_page',
                'title' => 'Platform',
                'description' => 'User search, tier management and verification',
                'url' => $this->safeRoute('hub.platform', '/hub/platform'),
                'icon' => 'server',
            ];
        }

        return $pages;
    }

    protected function safeRoute(string $name, string $fallback): string
    {
        return Route::has($name) ? route($name) : $fallback;
    }
}
