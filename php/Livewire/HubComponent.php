<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Livewire;

use Core\Tenant\Models\Workspace;
use Flux\Flux;
use Illuminate\Contracts\View\View;
use Livewire\Component;

/**
 * Shared hub component base for agentic Livewire screens.
 *
 * Example:
 *   final class FleetOverview extends HubComponent
 *   {
 *       protected function viewPath(): string
 *       {
 *           return __DIR__.'/../resources/views/livewire/agentic/fleet-overview.blade.php';
 *       }
 *   }
 */
abstract class HubComponent extends Component
{
    /**
     * Render the Livewire screen from its resolved Blade file.
     *
     * @example
     * $view = $component->render();
     */
    public function render(): View
    {
        return view()->file($this->viewPath());
    }

    /**
     * Resolve the current workspace identifier for the admin screen.
     *
     * @example
     * $workspaceId = $this->resolveWorkspaceId();
     */
    protected function resolveWorkspaceId(): int
    {
        try {
            return (int) (Workspace::query()->value('id') ?? 0);
        } catch (\Throwable) {
            return 0;
        }
    }

    /**
     * Abort the request when the current user lacks Hades access.
     *
     * @example
     * $this->checkHadesAccess();
     */
    protected function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    /**
     * Dispatch a toast notification through Flux or the fallback event bus.
     *
     * @example
     * $this->toast('Saved', 'Ledger refreshed.', 'success');
     */
    protected function toast(string $heading, string $text, string $variant): void
    {
        if (class_exists(Flux::class)) {
            Flux::toast(
                heading: $heading,
                text: $text,
                variant: $variant,
            );

            return;
        }

        $this->dispatch('notify', message: $text, variant: $variant);
    }

    /**
     * Resolve the Blade view file rendered by the Livewire screen.
     */
    abstract protected function viewPath(): string;
}
