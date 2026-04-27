<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Mod\Agentic\Website\Hub\Support\HubRouteNames;
use Core\Tenant\Services\WorkspaceService;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\On;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Workspaces')]
#[Layout('hub::admin.layouts.app')]
class Sites extends Component
{
    public string $search = '';

    protected WorkspaceService $workspaceService;

    public function boot(WorkspaceService $workspaceService): void
    {
        $this->workspaceService = $workspaceService;
    }

    #[On('workspace-changed')]
    public function refreshWorkspaceList(): void
    {
        unset($this->workspaces);
    }

    #[Computed]
    public function workspaces(): array
    {
        return $this->workspaceService->all();
    }

    #[Computed]
    public function filteredWorkspaces(): array
    {
        if ($this->search === '') {
            return $this->workspaces;
        }

        $query = mb_strtolower($this->search);

        return array_filter($this->workspaces, static function (array $workspace) use ($query): bool {
            return str_contains(mb_strtolower((string) ($workspace['name'] ?? '')), $query)
                || str_contains(mb_strtolower((string) ($workspace['slug'] ?? '')), $query)
                || str_contains(mb_strtolower((string) ($workspace['description'] ?? '')), $query);
        });
    }

    public function openWorkspace(string $slug): void
    {
        $this->redirect(HubRouteNames::url('sites.settings', ['workspace' => $slug], fallback: '/hub/workspaces/'.$slug), navigate: true);
    }

    public function render()
    {
        return view('hub::admin.sites');
    }
}
