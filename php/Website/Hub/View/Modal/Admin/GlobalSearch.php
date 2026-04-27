<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Mod\Agentic\Mod\Admin\Search\SearchProviderRegistry;
use Livewire\Attributes\Computed;
use Livewire\Attributes\On;
use Livewire\Component;

class GlobalSearch extends Component
{
    public bool $open = false;

    public string $query = '';

    public int $selectedIndex = 0;

    public array $recentSearches = [];

    protected int $maxRecentSearches = 5;

    protected SearchProviderRegistry $registry;

    public function boot(SearchProviderRegistry $registry): void
    {
        $this->registry = $registry;
    }

    public function mount(): void
    {
        $this->recentSearches = session('global_search.recent', []);
    }

    #[On('open-global-search')]
    public function openSearch(): void
    {
        $this->open = true;
        $this->query = '';
        $this->selectedIndex = 0;
    }

    public function closeSearch(): void
    {
        $this->open = false;
        $this->query = '';
        $this->selectedIndex = 0;
    }

    public function updatedQuery(): void
    {
        $this->selectedIndex = 0;
    }

    public function navigateUp(): void
    {
        if ($this->selectedIndex > 0) {
            $this->selectedIndex--;
        }
    }

    public function navigateDown(): void
    {
        if ($this->selectedIndex < count($this->flatResults) - 1) {
            $this->selectedIndex++;
        }
    }

    public function selectCurrent(): void
    {
        if (isset($this->flatResults[$this->selectedIndex])) {
            $this->navigateTo($this->flatResults[$this->selectedIndex]);
        }
    }

    /**
     * @param  array<string, mixed>  $result
     */
    public function navigateTo(array $result): void
    {
        $this->addToRecentSearches($result);
        $this->closeSearch();
        $this->dispatch('navigate-to-url', url: $result['url']);
    }

    public function navigateToRecent(int $index): void
    {
        if (! isset($this->recentSearches[$index])) {
            return;
        }

        $this->closeSearch();
        $this->dispatch('navigate-to-url', url: $this->recentSearches[$index]['url']);
    }

    public function clearRecentSearches(): void
    {
        $this->recentSearches = [];
        session()->forget('global_search.recent');
    }

    public function removeRecentSearch(int $index): void
    {
        if (! isset($this->recentSearches[$index])) {
            return;
        }

        array_splice($this->recentSearches, $index, 1);
        session(['global_search.recent' => $this->recentSearches]);
    }

    /**
     * @param  array<string, mixed>  $result
     */
    protected function addToRecentSearches(array $result): void
    {
        $this->recentSearches = array_values(array_filter(
            $this->recentSearches,
            static fn (array $recent): bool => $recent['id'] !== $result['id'] || $recent['type'] !== $result['type']
        ));

        array_unshift($this->recentSearches, [
            'id' => $result['id'],
            'title' => $result['title'],
            'subtitle' => $result['subtitle'] ?? '',
            'url' => $result['url'],
            'type' => $result['type'],
            'icon' => $result['icon'],
        ]);

        $this->recentSearches = array_slice($this->recentSearches, 0, $this->maxRecentSearches);
        session(['global_search.recent' => $this->recentSearches]);
    }

    #[Computed]
    public function results(): array
    {
        if (mb_strlen($this->query) < 2) {
            return [];
        }

        $user = auth()->user();
        $workspace = method_exists($user, 'defaultHostWorkspace') ? $user->defaultHostWorkspace() : null;

        return $this->registry->search($this->query, $user, $workspace);
    }

    #[Computed]
    public function flatResults(): array
    {
        return $this->registry->flattenResults($this->results);
    }

    #[Computed]
    public function hasResults(): bool
    {
        return $this->flatResults !== [];
    }

    #[Computed]
    public function showRecentSearches(): bool
    {
        return mb_strlen($this->query) < 2 && $this->recentSearches !== [];
    }

    public function render()
    {
        return view('hub::admin.global-search');
    }
}
