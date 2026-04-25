<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mod\Admin;

use Core\Mod\Agentic\Mod\Admin\Search\Contracts\SearchProvider;
use Core\Mod\Agentic\Mod\Admin\Search\SearchProviderRegistry;
use Core\Mod\Agentic\Mod\Admin\Search\SearchResult;

class SearchProviderRegistryTest extends AdminTestCase
{
    public function test_SearchProviderRegistry_register_Good(): void
    {
        $registry = new SearchProviderRegistry;
        $registry->register($this->makeProvider());

        $this->assertCount(1, $registry->providers());
    }

    public function test_SearchProviderRegistry_search_Good_groups_results(): void
    {
        $registry = new SearchProviderRegistry;
        $registry->register($this->makeProvider());

        $results = $registry->search('dash', $this->hadesUser);

        $this->assertArrayHasKey('pages', $results);
        $this->assertSame('Pages', $results['pages']['label']);
        $this->assertCount(2, $results['pages']['results']);
    }

    public function test_SearchProviderRegistry_fuzzyMatch_Ugly_handles_abbreviations(): void
    {
        $registry = new SearchProviderRegistry;

        $this->assertTrue($registry->fuzzyMatch('gs', 'Global Search'));
        $this->assertTrue($registry->fuzzyMatch('dbd', 'dashboard'));
        $this->assertFalse($registry->fuzzyMatch('zzz', 'dashboard'));
    }

    protected function makeProvider(): SearchProvider
    {
        return new class implements SearchProvider
        {
            public function search(string $query): array
            {
                return [
                    new SearchResult('1', 'admin_page', 'Dashboard', 'Hub landing page', '/hub', 'house'),
                    new SearchResult('2', 'admin_page', 'Usage', 'Usage page', '/hub/account/usage', 'chart-pie'),
                ];
            }

            public function name(): string
            {
                return 'Pages';
            }

            public function icon(): string
            {
                return 'rectangle-stack';
            }
        };
    }
}
