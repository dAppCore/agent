<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Mod\Agentic\Website\Hub\Support\HubRouteNames;
use Core\Tenant\Models\Workspace;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Site Settings')]
#[Layout('hub::admin.layouts.app')]
class SiteSettings extends Component
{
    public string $workspaceSlug = '';

    public string $tab = 'services';

    public function mount(string $workspace, ?string $tab = null): void
    {
        $this->workspaceSlug = $workspace;

        if ($tab !== null && in_array($tab, $this->allowedTabs(), true)) {
            $this->tab = $tab;
        }

        if ($this->workspace === null) {
            abort(404);
        }

        if (! $this->authorisedForWorkspace()) {
            abort(403);
        }
    }

    #[Computed]
    public function workspace(): ?Workspace
    {
        $user = auth()->user();

        if ($user === null) {
            return null;
        }

        if (method_exists($user, 'workspaces')) {
            return $user->workspaces()->where('slug', $this->workspaceSlug)->first();
        }

        return class_exists(Workspace::class)
            ? Workspace::query()->where('slug', $this->workspaceSlug)->first()
            : null;
    }

    #[Computed]
    public function tabs(): array
    {
        return collect($this->allowedTabs())->mapWithKeys(function (string $tab): array {
            return [$tab => [
                'label' => ucfirst($tab),
                'href' => HubRouteNames::url('sites.settings', ['workspace' => $this->workspaceSlug, 'tab' => $tab], fallback: '/hub/workspaces/'.$this->workspaceSlug.'/'.$tab),
            ]];
        })->all();
    }

    /**
     * @return array<int, string>
     */
    protected function allowedTabs(): array
    {
        return ['services', 'general', 'deployment', 'environment', 'ssl', 'backups', 'danger'];
    }

    protected function authorisedForWorkspace(): bool
    {
        $workspace = $this->workspace;

        if ($workspace === null) {
            return false;
        }

        if (auth()->user()?->isHades()) {
            return true;
        }

        $role = $workspace->pivot->role ?? null;

        return $role === null || in_array($role, ['owner', 'admin'], true);
    }

    public function render()
    {
        return view('hub::admin.site-settings');
    }
}
