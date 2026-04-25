<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Mod\Agentic\Website\Hub\Support\HubRouteNames;
use Core\Tenant\Services\WorkspaceService;
use Livewire\Attributes\On;
use Livewire\Component;

class WorkspaceSwitcher extends Component
{
    public array $workspaces = [];

    public array $current = [];

    public bool $open = false;

    public string $returnUrl = '';

    protected WorkspaceService $workspaceService;

    public function boot(WorkspaceService $workspaceService): void
    {
        $this->workspaceService = $workspaceService;
    }

    public function mount(): void
    {
        $this->refreshFromService();
        $this->returnUrl = url()->current();
    }

    #[On('workspace-activated')]
    public function refreshWorkspaces(): void
    {
        $this->refreshFromService();
    }

    public function switchWorkspace(string $slug): void
    {
        if (! $this->workspaceService->setCurrent($slug)) {
            return;
        }

        $this->refreshFromService();
        $this->open = false;

        $this->dispatch('workspace-changed', workspace: $slug);
        $this->redirect($this->returnUrl !== '' ? $this->returnUrl : HubRouteNames::url('dashboard', fallback: '/hub'));
    }

    protected function refreshFromService(): void
    {
        $this->workspaces = $this->workspaceService->all();
        $this->current = $this->workspaceService->current();
    }

    public function render()
    {
        return view('hub::admin.workspace-switcher');
    }
}
